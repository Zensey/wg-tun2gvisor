module github.com/zensey/wg-tun2gvisor

go 1.18

require (
	github.com/google/gopacket v1.1.19
	github.com/miekg/dns v1.1.49
	github.com/sirupsen/logrus v1.8.1
	golang.org/x/net v0.0.0-20220607020251-c690dde0001d
	golang.zx2c4.com/wireguard v0.0.0-20220601130007-6a08d81f6bc4
	
	gvisor.dev/gvisor v0.0.0-20220606171652-e8883fa9173a
	// gvisor.dev/gvisor v0.0.0-20211020211948-f76a604701b6
	//gvisor.dev/gvisor v0.0.0-20220121190119-4f2d380c8b55

	inet.af/tcpproxy v0.0.0-20220326234310-be3ee21c9fa0
)

require golang.org/x/time v0.0.0-20220411224347-583f2d630306

require (
	github.com/google/btree v1.1.1 // indirect
	golang.org/x/crypto v0.0.0-20220315160706-3147a52a75dd // indirect
	golang.org/x/mod v0.6.0-dev.0.20220106191415-9b9b3d81d5e3 // indirect
	golang.org/x/sys v0.0.0-20220520151302-bc2c85ada10a // indirect
	golang.org/x/tools v0.1.10 // indirect
	golang.org/x/xerrors v0.0.0-20220517211312-f3a8303e98df // indirect
	golang.zx2c4.com/wintun v0.0.0-20211104114900-415007cec224 // indirect
)

//replace golang.zx2c4.com/wireguard => ../../wireguard-go/
// replace gvisor.dev/gvisor => C:\Users\user\go\pkg\mod\gvisor.dev\gvisor@v0.0.0-20211020211948-f76a604701b6\
// replace gvisor.dev/gvisor => github.com/sagernet/gvisor v0.0.0-20220109124627-f8f67dadd776
replace gvisor.dev/gvisor => github.com/mysteriumnetwork/gvisor v0.0.0-20220615150015-d552b202473c
