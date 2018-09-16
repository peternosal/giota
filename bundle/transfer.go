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
	"github.com/iotaledger/giota/pow"
	"github.com/iotaledger/giota/signing"
	"github.com/iotaledger/giota/transaction"
	"github.com/iotaledger/giota/trinary"
	"math"
	"time"
)

const (
	// (3^27-1)/2
	MaxTimestampTrytes = "MMMMMMMMM"
)

type Transfers []Transfer

// Transfer descibes the data/value to transfer to an address.
type Transfer struct {
	Address signing.Address
	Value   int64
	Message trinary.Trytes
	Tag     trinary.Trytes
}

const SignatureMessageFragmentSizeTrinary = transaction.SignatureMessageFragmentTrinarySize / 3

// CreateBundle translates the transfer objects into a bundle consisting of all output transactions.
// If a transfer object's message exceeds the signature message fragment size (2187 trytes),
// additional transactions are added to the bundle to accustom the signature fragments.
func (trs Transfers) CreateBundle() (Bundle, []trinary.Trytes, int64) {
	var (
		bundle Bundle
		frags  []trinary.Trytes
		total  int64
	)
	for _, tr := range trs {
		numSignatures := 1

		// if the message is longer than 2187 trytes, increase the amount of transactions for the entry
		if len(tr.Message) > SignatureMessageFragmentSizeTrinary {
			// get total length, message / signature message fragment (2187 trytes)
			fragementsLength := int(math.Floor(float64(len(tr.Message)) / SignatureMessageFragmentSizeTrinary))
			numSignatures += fragementsLength

			// copy out every fragment
			for k := 0; k < fragementsLength; k++ {
				var fragment trinary.Trytes
				switch {
				// remainder
				case k == fragementsLength-1:
					fragment = tr.Message[k*SignatureMessageFragmentSizeTrinary:]
				default:
					fragment = tr.Message[k*SignatureMessageFragmentSizeTrinary : (k+1)*SignatureMessageFragmentSizeTrinary]
				}

				frags = append(frags, fragment)
			}
		} else {
			frags = append(frags, tr.Message)
		}

		// add output transaction(s) to the bundle for this transfer
		// slice the address in case the user provided one with a checksum
		bundle.AddEntry(numSignatures, tr.Address[:81], tr.Value, time.Now(), tr.Tag)

		// sum up the total value to transfer
		total += tr.Value
	}
	return bundle, frags, total
}

type AddressInfos []AddressInfo

// AddressInfo includes an address and its information for signing.
type AddressInfo struct {
	Seed     trinary.Trytes
	Index    uint
	Security signing.SecurityLevel
}

// Address makes an Address from an AddressInfo
func (a *AddressInfo) Address() (signing.Address, error) {
	return signing.NewAddress(a.Seed, a.Index, a.Security)
}

// Key makes a Key from an AddressInfo
func (a *AddressInfo) Key() (trinary.Trytes, error) {
	return signing.NewKey(a.Seed, a.Index, a.Security)
}

func DoPoW(trunkTx, branchTx trinary.Trytes, trytes []transaction.Transaction, mwm int64, pow pow.PowFunc) error {
	var prev trinary.Trytes
	var err error
	for i := len(trytes) - 1; i >= 0; i-- {
		switch {
		case i == len(trytes)-1:
			trytes[i].TrunkTransaction = trunkTx
			trytes[i].BranchTransaction = branchTx
		default:
			trytes[i].TrunkTransaction = prev
			trytes[i].BranchTransaction = trunkTx
		}

		timestamp := trinary.IntToTrits(time.Now().UnixNano()/1000000, transaction.TimestampTrinarySize).Trytes()
		trytes[i].AttachmentTimestamp = timestamp
		trytes[i].AttachmentTimestampLowerBound = ""
		trytes[i].AttachmentTimestampUpperBound = MaxTimestampTrytes

		trytes[i].Nonce, err = pow(trytes[i].Trytes(), int(mwm))
		if err != nil {
			return err
		}

		prev = trytes[i].Hash()
	}
	return nil
}
