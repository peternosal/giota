/*
MIT License

Copyright (c) 2017 Shinya Yagyu

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/
package bundle

import (
	"github.com/iotaledger/giota"
	"github.com/iotaledger/giota/bundle"
	"github.com/iotaledger/giota/pow"
	"github.com/iotaledger/giota/signing"
	"github.com/iotaledger/giota/trinary"
	"os"
	"testing"
)

var (
	seed             trinary.Trytes
	skipTransferTest = false
)

func init() {
	ts := os.Getenv("TRANSFER_TEST_SEED")
	if ts == "" {
		skipTransferTest = true
		return
	}

	s, err := trinary.ToTrytes(ts)
	if err != nil {
		skipTransferTest = true
	} else {
		seed = s
	}
}

func TestTransfer1(t *testing.T) {
	if skipTransferTest {
		t.Skip("transfer test skipped because a valid $TRANSFER_TEST_SEED was not specified")
	}

	var (
		err  error
		adr  signing.Address
		adrs []signing.Address
	)

	for i := 0; i < 5; i++ {
		api := giota.NewAPI(giota.RandomNode(), nil)
		adr, adrs, err = GetUsedAddress(api, seed, 2)
		if err == nil {
			break
		}
	}

	if err != nil {
		t.Error(err)
	}

	t.Log(adr, adrs)
	if len(adrs) < 1 {
		t.Error("GetUsedAddress is incorrect")
	}

	var bal giota.Balances
	for i := 0; i < 5; i++ {
		api := giota.NewAPI(giota.RandomNode(), nil)
		bal, err = GetInputs(api, seed, 0, 10, 1000, 2)
		if err == nil {
			break
		}
	}

	if err != nil {
		t.Error(err)
	}

	t.Log(bal)
	if len(bal) < 1 {
		t.Error("GetInputs is incorrect")
	}
}

// nolint: gocyclo
func TestTransfer2(t *testing.T) {
	if skipTransferTest {
		t.Skip("transfer test skipped because a valid $TRANSFER_TEST_SEED was not specified")
	}

	var err error
	trs := []Transfer{
		Transfer{
			Address: "KTXFP9XOVMVWIXEWMOISJHMQEXMYMZCUGEQNKGUNVRPUDPRX9IR9LBASIARWNFXXESPITSLYAQMLCLVTL9QTIWOWTY",
			Value:   20,
			Tag:     "MOUDAMEPO",
		},
	}

	var bdl bundle.Bundle
	for i := 0; i < 5; i++ {
		api := giota.NewAPI(giota.RandomNode(), nil)
		bdl, err = PrepareTransfers(api, seed, trs, nil, "", 2)
		if err == nil {
			break
		}
	}

	if err != nil {
		t.Error(err)
	}

	if len(bdl) < 3 {
		for _, tx := range bdl {
			t.Log(tx.Trytes())
		}
		t.Fatal("PrepareTransfers is incorrect len(bdl)=", len(bdl))
	}

	if err = bdl.IsValid(); err != nil {
		t.Error(err)
	}

	name, pow := pow.GetBestPoW()
	t.Log("using PoW: ", name)

	for i := 0; i < 5; i++ {
		api := giota.NewAPI(giota.RandomNode(), nil)
		bdl, err = Send(api, seed, 2, trs, 18, pow)
		if err == nil {
			break
		}
	}

	if err != nil {
		t.Error(err)
	}

	for _, tx := range bdl {
		t.Log(tx.Trytes())
	}
}
