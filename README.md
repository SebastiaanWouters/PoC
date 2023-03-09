# PoC
Proof of Computation is a consensus algorithm that utilizes TEEs (Trusted Execution Environments)
to replace the 'wasted computation' in PoW with 'useful computation'.

This repositry contains an implementation of a node and a worker to demonstrate "Proof of Computation" in combination
with a blockchain and local P2P communication. Code is written in Go, compiled with EGo by Edgeless Systems.

# Create A Reproducible Build

Execute the listed commands in the worker directory to create an executable in a docker container and verify the source code,
the output of uniqueid should be: 3361447737af78e8f8ff9944a883dc9bef7b6f801c55c031bccfdc3ff82f9c89

Tested using Docker version 20.10.23, build 7155243

1. openssl genrsa -out private.pem -3 3072

2. sudo DOCKER_BUILDKIT=1 docker build --secret id=signingkey,src=private.pem -o. .

3. ego uniqueid worker

# Non Reproducible Build

Use ego-go build together with ego sign and ego run to create and run the worker inside of an enclave without taking advantage of reproducible builds.