package giota

import "errors"

// Address represents an address without a checksum.
// Don't type cast, use ToAddress instead to check validity.
type Address Trytes

// Error types for address
var (
	ErrInvalidAddressTrytes = errors.New("addresses without checksum are 81 trytes in length")
	ErrInvalidAddressTrits  = errors.New("addresses without checksum are 243 trits in length")
	ErrInvalidChecksum = errors.New("checksum doesn't match address")
)

// NewAddress generates a new address from seed without checksum
func NewAddress(seed Trytes, index, security int) (Address, error) {
	k, err := newKeyTrits(seed, index, security)
	if err != nil {
		return "", err
	}

	dg, err := Digests(k)
	if err != nil {
		return "", err
	}

	addr, err := addressFromDigests(dg)
	if err != nil {
		return "", err
	}

	tryt := addr.Trytes()
	return tryt.ToAddress()
}

// NewAddresses generates new count addresses from seed without a checksum
func NewAddresses(seed Trytes, start, count, security int) ([]Address, error) {
	as := make([]Address, count)

	var err error
	for i := 0; i < count; i++ {
		as[i], err = NewAddress(seed, start+i, security)
		if err != nil {
			return nil, err
		}
	}
	return as, nil
}


// addressFromDigests calculates the address from the given digests
func addressFromDigests(digests Trits) (Trits, error) {
	k := NewKerl()
	k.Absorb(digests)
	return k.Squeeze(HashSize)
}

// ToAddress converts string to address, and checks the validity
func ToAddress(t string) (Address, error) {
	return Trytes(t).ToAddress()
}

// ToAddress convert trytes (with and without checksum) to address and checks the validity
func (t Trytes) ToAddress() (Address, error) {
	if len(t) == 90 {
		t = t[:81]
	}

	a := Address(t)
	if err := a.IsValid(); err != nil {
		return "", err
	}

	// validate the checksum
	if len(t) == 90 {
		if err := a.IsValidChecksum(t[81:]); err != nil {
			return "", err
		}
	}

	return a, nil
}

// IsValid returns nil if address is valid
func (a Address) IsValid() error {
	if !(len(a) == 81) {
		return ErrInvalidAddressTrytes
	}

	return Trytes(a).IsValid()
}

func(a Address) IsValidChecksum(checksum Trytes) error {
	checksumFromAddress, err := a.Checksum()
	if err != nil {
		return err
	}
	if checksumFromAddress != checksum {
		return ErrInvalidChecksum
	}
	return nil
}

// Checksum returns checksum trytes
func (a Address) Checksum() (Trytes, error) {
	if len(a) != 81 {
		return "", ErrInvalidAddressTrytes
	}

	checksumHash, err := a.ChecksumHash()
	if err != nil {
		return "", err
	}
	return checksumHash[81-9 : 81], nil
}

// ChecksumHash hashes the address and returns the 81 trytes long checksum hash
func (a Address) ChecksumHash() (Trytes, error) {
	k := NewKerl()
	t := Trytes(a).Trits()
	k.Absorb(t)
	h, err := k.Squeeze(HashSize)
	if err != nil {
		return "", err
	}
	return h.Trytes(), nil
}

// WithChecksum returns the address together with the checksum. (90 trytes)
func (a Address) WithChecksum() (Trytes, error) {
	if len(a) != 81 {
		return "", ErrInvalidAddressTrytes
	}

	cu, err := a.Checksum()
	if err != nil {
		return "", err
	}

	return Trytes(a) + cu, nil
}
