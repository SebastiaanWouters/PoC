# PoC
Proof of Computation is a consensus algorithm that utilizes TEEs (Trusted Execution Environments)
to replace the 'wasted computation' in PoW with 'useful computation'.

This repositry contains an implementation of a node and a worker to demonstrate "Proof of Computation" in combination
with a blockchain and local P2P communication. Code is written in Go, compiled with EGo by Edgeless Systems.

# Create A Reproducible Build

openssl genrsa -out private.pem -3 3072
DOCKER_BUILDKIT=1 docker build --secret id=signingkey,src=private.pem -o. .
ego uniqueid worker