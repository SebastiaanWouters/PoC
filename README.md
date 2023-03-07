# PoC
Proof of Computation is a consensus algorithm that utilizes TEEs (Trusted Execution Environments)
to replace the 'wasted computation' in PoW with 'useful computation'.

This repositry contains an implementation of a node and a worker to demonstrate "Proof of Computation" in combination
with a blockchain and local P2P communication. Code is written in Go, compiled with EGo by Edgeless Systems.

# Create A Reproducible Build

Execute the listed commands to create an executable and verify the source code,
the output of uniqueid should be: 71ae46f315cdfb4bc6d3f45919d7a72f2d09c23d37d42d8e7f3a835f587b8117
Tested using Docker version 20.10.23, build 7155243

1. openssl genrsa -out private.pem -3 3072

2. sudo DOCKER_BUILDKIT=1 docker build --secret id=signingkey,src=private.pem -o. .

3. ego uniqueid worker