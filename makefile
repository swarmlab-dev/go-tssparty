cli:
	cd cli && go build -o tssparty .

aar:
	gomobile bind -target=android github.com/swarmlab-dev/go-tss/tssparty 

all: cli aar