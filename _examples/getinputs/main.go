package main

import (
	"errors"
	"fmt"
	"net/http"
	"syscall"
	"time"

	"github.com/iotaledger/giota"
	"github.com/iotaledger/giota/trinary"

	"golang.org/x/crypto/ssh/terminal"
)

const Host = "http://node03.iotatoken.nl:15265"

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	fmt.Print("input your seed: ")
	api := giota.NewAPI(Host, &client)
	seed, err := terminal.ReadPassword(int(syscall.Stdin))
	must(err)

	seedT, err := trinary.ToTrytes(string(seed))
	must(err)

	fmt.Print("\nhow many addresses should we check: ")
	var offset int
	n, err := fmt.Scanf("%d\n", &offset)
	if err != nil || n < 1 {
		panic(errors.New("expting integer offset"))
	}

	fmt.Print("which security level should the addresses be generated at (0, 1, 2; default is 2): ")
	var slevel int
	n, err = fmt.Scanf("%d\n", &slevel)
	if err != nil || n < 1 || slevel < 0 || slevel > 2 {
		slevel = 2
	}

	println("Getting balances")
	// GetInputs(API, seed, start index, end index, threshold, security level)
	inputs, err := api.GetInputs(seedT, 0, offset, 0, slevel)
	must(err)

	fmt.Println(inputs)
}
