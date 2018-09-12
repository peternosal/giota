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
	"errors"
	"fmt"
	"github.com/iotaledger/giota/curl"
	"github.com/iotaledger/giota/kerl"
	"github.com/iotaledger/giota/signing"
	"github.com/iotaledger/giota/trinary"
	"github.com/iotaledger/giota/tx"
	"time"
)

func pad(orig trinary.Trytes, size int) trinary.Trytes {
	out := make([]byte, size)
	copy(out, []byte(orig))

	for i := len(orig); i < size; i++ {
		out[i] = '9'
	}
	return trinary.Trytes(out)
}

// Bundle is transactions that are bundled (grouped) together when creating a transfer.
type Bundle []tx.Tx

// Add adds a bundle to bundle slice. Elements which are not specified are filled with
// zeroed trits.
func (bundle *Bundle) Add(num int, address signing.Address, value int64, timestamp time.Time, tag trinary.Trytes) {
	if tag == "" {
		tag = curl.EmptyHash[:27]
	}

	for i := 0; i < num; i++ {
		var v int64

		if i == 0 {
			v = value
		}

		b := tx.Tx{
			SignatureMessageFragment:      signing.EmptySig,
			Address:                       address,
			Value:                         v,
			ObsoleteTag:                   pad(tag, tx.TagTrinarySize/3),
			Timestamp:                     timestamp,
			CurrentIndex:                  int64(len(*bundle) - 1),
			LastIndex:                     0,
			Bundle:                        curl.EmptyHash,
			TrunkTx:                       curl.EmptyHash,
			BranchTx:                      curl.EmptyHash,
			Tag:                           pad(tag, tx.TagTrinarySize/3),
			AttachmentTimestamp:           curl.EmptyHash,
			AttachmentTimestampLowerBound: curl.EmptyHash,
			AttachmentTimestampUpperBound: curl.EmptyHash,
			Nonce:                         curl.EmptyHash,
		}
		*bundle = append(*bundle, b)
	}
}

// Finalize filled sigs, bundlehash, and indices elements in bundle.
func (bundle Bundle) Finalize(sig []trinary.Trytes) {
	h := bundle.GetValidHash()

	for i := range bundle {
		if len(sig) > i && sig[i] != "" {
			bundle[i].SignatureMessageFragment = pad(sig[i], tx.SignatureMessageFragmentTrinarySize/3)
		}

		bundle[i].CurrentIndex = int64(i)
		bundle[i].LastIndex = int64(len(bundle) - 1)
		bundle[i].Bundle = h
	}
}

// Hash calculates hash of Bundle.
func (bundle Bundle) Hash() trinary.Trytes {
	k := kerl.NewKerl()
	buf := make(trinary.Trits, 243+81*3)

	for i, b := range bundle {
		getTritsToHash(buf, &b, i, len(bundle))
		k.Absorb(buf)
	}

	// TODO: handle error!
	h, _ := k.Squeeze(curl.HashSize)
	return h.Trytes()
}

// GetValidHash calculates hash of Bundle and increases ObsoleteTag value
// until normalized hash doesn't have any 13
func (bundle Bundle) GetValidHash() trinary.Trytes {
	k := kerl.NewKerl()
	hashedLen := tx.BundleTrinaryOffset - tx.AddressTrinaryOffset

	buf := make(trinary.Trits, hashedLen*len(bundle))
	for i, b := range bundle {
		getTritsToHash(buf[i*hashedLen:], &b, i, len(bundle))
	}

	for {
		k.Absorb(buf)
		hashTrits, _ := k.Squeeze(curl.HashSize)
		h := hashTrits.Trytes()
		n := h.Normalize()
		valid := true

		for _, v := range n {
			if v == 13 {
				valid = false
				break
			}
		}

		offset := tx.ObsoleteTagTrinaryOffset - tx.AddressTrinaryOffset

		if valid {
			bundle[0].ObsoleteTag = buf[offset:offset+tx.ObsoleteTagTrinarySize].Trytes()
			return h
		}

		k.Reset()
		trinary.IncTrits(buf[offset : offset+tx.ObsoleteTagTrinarySize])
	}
}

func getTritsToHash(buf trinary.Trits, b *tx.Tx, i, l int) {
	copy(buf, trinary.Trytes(b.Address).Trits())
	copy(buf[243:], trinary.Int2Trits(b.Value, tx.ValueTrinarySize))
	copy(buf[243+81:], b.ObsoleteTag.Trits())
	copy(buf[243+81+81:], trinary.Int2Trits(b.Timestamp.Unix(), tx.TimestampTrinarySize))
	copy(buf[243+81+81+27:], trinary.Int2Trits(int64(i), tx.CurrentIndexTrinarySize))   //CurrentIndex
	copy(buf[243+81+81+27+27:], trinary.Int2Trits(int64(l-1), tx.LastIndexTrinarySize)) //LastIndex
}

// Categorize categorizes a list of transfers into sent and received. It is important to
// note that zero value transfers (which for example, are being used for storing
// addresses in the Tangle), are seen as received in this function.
func (bundle Bundle) Categorize(adr signing.Address) (send Bundle, received Bundle) {
	send = make(Bundle, 0, len(bundle))
	received = make(Bundle, 0, len(bundle))

	for _, b := range bundle {
		switch {
		case b.Address != adr:
			continue
		case b.Value >= 0:
			received = append(received, b)
		default:
			send = append(send, b)
		}
	}
	return
}

// IsValid checks the validity of Bundle.
// It checks that total balance==0 and that its has a valid signature.
// The caller must call Finalize() beforehand.
// nolint: gocyclo
func (bundle Bundle) IsValid() error {
	var total int64
	sigs := make(map[signing.Address][]trinary.Trytes)
	for index, b := range bundle {
		total += b.Value

		switch {
		case b.CurrentIndex != int64(index):
			return fmt.Errorf("CurrentIndex of index %d is not correct", b.CurrentIndex)
		case b.LastIndex != int64(len(bundle)-1):
			return fmt.Errorf("LastIndex of index %d is not correct", b.CurrentIndex)
		case b.Value >= 0:
			continue
		}

		sigs[b.Address] = append(sigs[b.Address], b.SignatureMessageFragment)

		// Find the subsequent txs with the remaining signature fragment
		for i := index; i < len(bundle)-1; i++ {
			tx := bundle[i+1]

			// Check if new tx is part of the signature fragment
			if tx.Address == b.Address && tx.Value == 0 {
				sigs[tx.Address] = append(sigs[tx.Address], tx.SignatureMessageFragment)
			}
		}
	}

	// Validate the signatures
	h := bundle.Hash()
	for adr, sig := range sigs {
		if !signing.IsValidSig(adr, sig, h) {
			return errors.New("invalid signature")
		}
	}

	if total != 0 {
		return errors.New("total balance of Bundle is not 0")
	}

	return nil
}

func (bundle Bundle) SignInputs(inputs []AddressInfo) error {
	//  Get the normalized bundle hash
	nHash := bundle.Hash().Normalize()

	// SIGNING OF INPUTS
	// Here we do the actual signing of the inputs. Iterate over all bundle transactions,
	// find the inputs, get the corresponding private key, and calculate signatureFragment
	for i, bd := range bundle {
		if bd.Value >= 0 {
			continue
		}

		// Get the corresponding keyIndex and security of the address
		var ai AddressInfo
		for _, in := range inputs {
			adr, err := in.Address()
			if err != nil {
				return err
			}

			if adr == bd.Address {
				ai = in
				break
			}
		}

		// Get corresponding private key of the address
		key, err := ai.Key()
		if err != nil {
			return err
		}

		// Calculate the new signatureFragment with the first bundle fragment
		bundle[i].SignatureMessageFragment = signing.Sign(nHash[:27], key[:6561/3])

		// if user chooses higher than 27-tryte security
		// for each security level, add an additional signature
		for j := 1; j < ai.Security; j++ {
			//  Because the signature is > 2187 trytes, we need to find the subsequent
			// transaction to add the remainder of the signature same address as well
			// as value = 0 (as we already spent the input)
			if bundle[i+j].Address == bd.Address && bundle[i+j].Value == 0 {
				//  Calculate the new signature
				nfrag := signing.Sign(nHash[(j%3)*27:(j%3)*27+27], key[6561*j/3:(j+1)*6561/3])
				//  Convert signature to trytes and assign it again to this bundle entry
				bundle[i+j].SignatureMessageFragment = nfrag
			}
		}
	}
	return nil
}
