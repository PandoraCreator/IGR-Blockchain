"""
Interact with a Fabric 2.x network (custom-network / Docker) using the `peer` CLI.

Hyperledger does not publish a first-party Python Fabric Gateway SDK; official
client APIs are Go, Node, and Java. This module runs the `peer` binary (same
flow as `scripts/custom-network.sh`) with TLS and MSP paths aligned to
`network/crypto-config/crypto-config/`.

Requires `./scripts/bootstrap-fabric.sh` so `bin/bin/peer` exists, and a
running network with channel + chaincode deployed.
"""

from __future__ import annotations

import json
import os
import shutil
import subprocess
from dataclasses import dataclass, field
from pathlib import Path
from typing import List, Optional, Sequence


@dataclass
class NetworkPaths:
    """Cryptographic and config paths for the lab network (cryptogen output)."""

    repo_root: Path
    msp_id: str = "Org1MSP"
    admin_msp: Path = field(init=False)  # Admin@org1.example.com
    org1_peer0_tls_ca: Path = field(init=False)
    org2_peer0_tls_ca: Path = field(init=False)
    orderer_tls_ca: Path = field(init=False)
    fabric_cfg: Path = field(init=False)

    def __post_init__(self) -> None:
        root = self.repo_root
        p = (
            root
            / "network/crypto-config/crypto-config/peerOrganizations/org1.example.com"
        )
        self.admin_msp = p / "users/Admin@org1.example.com/msp"
        self.org1_peer0_tls_ca = p / "peers/peer0.org1.example.com/tls/ca.crt"
        p2 = (
            root
            / "network/crypto-config/crypto-config/peerOrganizations/org2.example.com"
        )
        self.org2_peer0_tls_ca = p2 / "peers/peer0.org2.example.com/tls/ca.crt"
        self.orderer_tls_ca = (
            root
            / "network/crypto-config/crypto-config/ordererOrganizations/example.com"
            / "orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem"
        )
        self.fabric_cfg = root / "production/fabric-config"


@dataclass
class Endorser:
    address: str
    tls_root_cert: Path


def _default_endorsers(paths: NetworkPaths) -> List[Endorser]:
    return [
        Endorser("peer0.org1.example.com:7051", paths.org1_peer0_tls_ca),
        Endorser("peer0.org2.example.com:9051", paths.org2_peer0_tls_ca),
    ]


class FabricPeerClient:
    """
    Read (evaluate/query) and write (invoke + commit) through the `peer` CLI.

    Defaults match `scripts/env.sh` and the asset-transfer-basic chaincode
    (chaincode name `basic`, channel `mychannel`).
    """

    def __init__(
        self,
        paths: NetworkPaths,
        *,
        channel: str = "mychannel",
        chaincode_name: str = "basic",
        peer_address: str = "peer0.org1.example.com:7051",
        orderer_address: str = "orderer.example.com:7050",
        orderer_tls_hostname: str = "orderer.example.com",
        peer_binary: Optional[Path] = None,
        endorsers: Optional[Sequence[Endorser]] = None,
    ) -> None:
        self.paths = paths
        self.channel = channel
        self.chaincode_name = chaincode_name
        self.peer_address = peer_address
        self.orderer_address = orderer_address
        self.orderer_tls_hostname = orderer_tls_hostname
        self.endorsers: List[Endorser] = (
            list(endorsers) if endorsers is not None else _default_endorsers(paths)
        )
        if peer_binary is not None:
            self._peer = peer_binary
        else:
            guess = paths.repo_root / "bin" / "bin" / "peer"
            self._peer = guess if guess.is_file() else Path(shutil.which("peer") or "peer")

    def _peer_env(self) -> dict:
        return {
            **os.environ,
            "FABRIC_CFG_PATH": str(self.paths.fabric_cfg),
            "CORE_PEER_LOCALMSPID": self.paths.msp_id,
            "CORE_PEER_TLS_ENABLED": "true",
            "CORE_PEER_MSPCONFIGPATH": str(self.paths.admin_msp),
            "CORE_PEER_TLS_ROOTCERT_FILE": str(self.org1_peer_tls_for_client()),
            "CORE_PEER_ADDRESS": self.peer_address,
        }

    def org1_peer_tls_for_client(self) -> Path:
        return self.paths.org1_peer0_tls_ca

    def _run_peer(self, args: List[str], *, check: bool = True) -> subprocess.CompletedProcess:
        if not self._peer.is_file() and not shutil.which(str(self._peer)):
            raise FileNotFoundError(
                f"peer binary not found at {self._peer}. Run ./scripts/bootstrap-fabric.sh"
            )
        proc = subprocess.run(
            [str(self._peer), *args],
            env=self._peer_env(),
            check=False,
            text=True,
            capture_output=True,
        )
        if check and proc.returncode != 0:
            err = (proc.stderr or proc.stdout or "").strip()
            raise RuntimeError(
                f"peer exited {proc.returncode}: {err}\ncommand: {args[:6]}..."
            )
        return proc

    def evaluate(self, chaincode_fn: str, args: List[str]) -> str:
        """
        Read-only query (no ledger update). Maps to: peer chaincode query
        """
        ctor = json.dumps(
            {"Args": [chaincode_fn, *args]}, separators=(",", ":")
        )
        proc = self._run_peer(
            [
                "chaincode",
                "query",
                "-C",
                self.channel,
                "-n",
                self.chaincode_name,
                "-c",
                ctor,
            ]
        )
        if proc.returncode != 0:
            raise RuntimeError(
                f"query failed: {proc.stderr or proc.stdout}\nargs={ctor}"
            )
        # `peer chaincode query` writes the result to stdout.
        return proc.stdout.strip()

    def submit(
        self,
        chaincode_fn: str,
        args: List[str],
        *,
        wait_for_event: bool = True,
    ) -> str:
        """
        Submit a transaction (writes). Maps to: peer chaincode invoke with
        endorsements from default Org1+Org2 peers, orderer, and optional
        --waitForEvent.
        """
        # Match scripts/custom-network.sh invoke payload style for Node contract.
        invoke_payload = json.dumps(
            {"function": chaincode_fn, "Args": list(args)},
            separators=(",", ":"),
        )
        cmd: List[str] = [
            "chaincode",
            "invoke",
            "-o",
            self.orderer_address,
            "--ordererTLSHostnameOverride",
            self.orderer_tls_hostname,
            "--tls",
            "--cafile",
            str(self.paths.orderer_tls_ca),
        ]
        if wait_for_event:
            cmd.append("--waitForEvent")
        cmd += [
            "-C",
            self.channel,
            "-n",
            self.chaincode_name,
        ]
        for e in self.endorsers:
            cmd += [
                "--peerAddresses",
                e.address,
                "--tlsRootCertFiles",
                str(e.tls_root_cert),
            ]
        cmd += ["-c", invoke_payload]
        proc = self._run_peer(cmd)
        if proc.returncode != 0:
            raise RuntimeError(
                f"invoke failed: {proc.stderr or proc.stdout}\n{invoke_payload=}"
            )
        # `peer chaincode invoke` logs the human-readable result on stderr (stdout is empty).
        return (proc.stdout or proc.stderr).strip()

    # --- sample chaincode (asset-transfer-basic) helpers ---

    def read_asset(self, asset_id: str) -> str:
        return self.evaluate("ReadAsset", [asset_id])

    def create_asset(
        self,
        asset_id: str,
        color: str,
        size: str,
        owner: str,
        appraised_value: str,
    ) -> str:
        return self.submit(
            "CreateAsset",
            [asset_id, color, size, owner, appraised_value],
        )
