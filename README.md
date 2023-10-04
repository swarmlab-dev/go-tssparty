# go-tssparty

This project `go-tssparty` contains a library and a command line tool that is essentially a wrapper around the bnb-chain mpc-tss implementation. This library facilitates the mpc-tss `keygen`, `signing` and `resharing` ceremonies. It uses the  [go-partybus](https://github.com/swarmlab-dev/go-partybus) transport method to broadcast message between threshold perticipants.

## Command line Usage

We assume that an instance of [go-partybus](https://github.com/swarmlab-dev/go-partybus) is already deployed and accessible. by default it is assumed to be available on `127.0.0.1:8080`.

### mpc-tss keygen ceremony

On three different terminals, use the following command to start the keygeneration ceremony:

```
$ ./cli keygen --eddsa -s test-keygen-1234

{ ..#KEYSHARE#.. }

```

By default, the keygen uses ecdsa and expect a 3 party share and a threshold of 2. Those parameters can be tune with `--eddsa` to use eddsa instead of ecdsa and  `-n` and `-t` respectively for the number of share and threshold. 

The argument `-s test-keygen-1234` is the name of the party room on the partybus server and must be the same for all participant. Once all participants are connected to the party room, the keygen ceremony starts and ends with each party outputing its share as a json file.

### mpc-tss keygen ceremony

On three different terminals, use the following command to start the keygeneration ceremony:

```
$ ./cli signing --eddsa -s test-signing-1234 -k '{ ..#KEYSHARE#.. }' -m "hello world"
```

Signing works similarly than keygen. The key share must be provided as input with the option `-k`. The message to sign must be the same on all perticipant with the option `-m`.

