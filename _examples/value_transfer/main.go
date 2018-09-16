package main

import (
	"github.com/iotaledger/giota"
	"github.com/iotaledger/giota/bundle"
	"github.com/iotaledger/giota/pow"
	"github.com/iotaledger/giota/signing"
	"github.com/iotaledger/giota/trinary"
	"github.com/iotaledger/giota/units"
	"net/http"
)

const iriEndpoint = "https://trinity.iota-tangle.io:14265"
const seed = "QKFIKNNOLHEWDEATLDRTIQYTMJUBQQGIXFBJUQRIFYXVBIUSOGNIBCAKEDCWBKGVPQODZVQSWUVFGLJ9M"

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	// create an API instance
	api := giota.NewAPI(iriEndpoint, http.DefaultClient)

	// our target address
	targetAddrTrinary := trinary.Trytes("DLXGUQYGLC9HZXNVLEKPXJYVJUSNXJGOKYJLAXETSN9QLIPGKTMYNDUZYNHQFTWJJBIZRGDSJITXAKWCWVZWVRMLID")
	targetAddr, err := signing.ToAddress(targetAddrTrinary) // if the addr contains a checksum it will be thrown away
	must(err)

	// message for the recipient of the transaction
	msgTrytes, err := trinary.ASCIIToTrytes("this transaction was made via the Go IOTA library")
	must(err)

	// create a new transfer object representing our value transfer
	transfers := bundle.Transfers{
		{
			Address: targetAddr,
			Value: int64(units.ConvertUnits(1, units.Ki, units.I)), // 1000 iotas
			Message: msgTrytes,
			Tag: trinary.Trytes("GIOTALIBRARY"), // gets automatically padded to 27 trytes
		},
	}

	// assume we have enough funds at address with index 0
	inputs := bundle.AddressInfos{
		{Seed: seed, Index: 0, Security: 2},
	}

	// compute remainder address for the transfer
	remainderAddr, err := bundle.AddressInfo{Seed: seed, Index: 1, Security: 2}.Address()
	must(err)

	// prepares the transfers by creating a bundle with the given output transaction (made from the transfer objects)
	// and input transactions from the given input address infos. in case not the entire input is spent to the
	// defined transfers, the remainder is sent to the given remainder address.
	// It also automatically checks whether the given input addresses have enough funds for the transfer.
	bundle, err := api.PrepareTransfers(seed, transfers, inputs, remainderAddr, 2)
	must(err)

	// at this point it is good practice to check whether the destination address was already spent from


	// at this point contains input and output transactions and is signed
	// now we need to first select two tips to approve and then do proof of work
	// we can do this in one call with SendTrytes() which does:
	// 1. select two tips
	// 2. create an attachToTangleRequest to the remote node or do PoW locally if powFunc is supplied
	// 3. broadcast the bundle to the network
	// 4. do a storeTransaction call to the connected node
	_, powFunc := pow.GetBestPoW()
	err = api.SendTrytes(3, bundle, 14, powFunc)
	must(err)

}
