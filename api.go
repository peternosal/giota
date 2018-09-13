/*
MIT License

Copyright (c) 2016 Sascha Hanse
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

package giota

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/iotaledger/giota/bundle"
	"github.com/iotaledger/giota/curl"
	"github.com/iotaledger/giota/pow"
	"github.com/iotaledger/giota/signing"
	"github.com/iotaledger/giota/transaction"
	"github.com/iotaledger/giota/trinary"

	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// PublicNodes is a list of known public nodes from http://iotasupport.com/lightwallet.shtml.
var (
	PublicNodes = []string{
		"http://service.iotasupport.com:14265",
		"http://eugene.iota.community:14265",
		"http://eugene.iotasupport.com:14999",
		"http://eugeneoldisoft.iotasupport.com:14265",
		"http://mainnet.necropaz.com:14500",
		"http://iotatoken.nl:14265",
		"http://iota.digits.blue:14265",
		"http://wallets.iotamexico.com:80",
		"http://5.9.137.199:14265",
		"http://5.9.118.112:14265",
		"http://5.9.149.169:14265",
		"http://88.198.230.98:14265",
		"http://176.9.3.149:14265",
		"http://iota.bitfinex.com:80",
	}
)

var (
	ErrInvalidTailTransactionHash = errors.New("the given transaction hash is not a trail transaction hash")
	ErrTxNotFound                 = errors.New("couldn't find transaction via getTrytes")
	ErrTxNotFoundInInclusionState = errors.New("couldn't find transactions in inclusion state call")
)

// RandomNode returns a random node from PublicNodes. If local IRI exists, return
// localhost address.
func RandomNode() string {
	api := NewAPI("", nil)
	_, err := api.GetNodeInfo()
	if err == nil {
		return api.endpoint
	}

	b := make([]byte, 1)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return PublicNodes[int(b[0])%len(PublicNodes)]
}

// API is for calling APIs.
type API struct {
	client   *http.Client
	endpoint string
}

// NewAPI takes an (optional) endpoint and optional http.Client and returns
// an API struct. If an empty endpoint is supplied, then "http://localhost:14265"
// is used.
func NewAPI(endpoint string, c *http.Client) *API {
	if c == nil {
		c = http.DefaultClient
	}

	if endpoint == "" {
		endpoint = "http://localhost:14265/"
	}

	return &API{client: c, endpoint: endpoint}
}

func handleError(err *ErrorResponse, err1, err2 error) error {
	switch {
	case err.Error != "":
		return errors.New(err.Error)
	case err.Exception != "":
		return errors.New(err.Exception)
	case err1 != nil:
		return err1
	}

	return err2
}

func (api *API) do(cmd interface{}, out interface{}) error {
	b, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	rd := bytes.NewReader(b)

	req, err := http.NewRequest("POST", api.endpoint, rd)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-IOTA-API-Version", "1")
	resp, err := api.client.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		if err = resp.Body.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		errResp := &ErrorResponse{}
		err = json.Unmarshal(bs, errResp)
		return handleError(errResp, err, fmt.Errorf("http status %d while calling API", resp.StatusCode))
	}

	if bytes.Contains(bs, []byte(`"error"`)) || bytes.Contains(bs, []byte(`"exception"`)) {
		errResp := &ErrorResponse{}
		err = json.Unmarshal(bs, errResp)
		return handleError(errResp, err, fmt.Errorf("unknown error occured while calling API"))
	}

	if out == nil {
		return nil
	}
	return json.Unmarshal(bs, out)
}

// ErrorResponse is for an exception occurring while calling API.
type ErrorResponse struct {
	Error     string `json:"error"`
	Exception string `json:"exception"`
}

// GetNodeInfoRequest is for GetNodeInfo API request.
type GetNodeInfoRequest struct {
	Command string `json:"command"`
}

// GetNodeInfoResponse is for GetNode API response.
type GetNodeInfoResponse struct {
	AppName                            string         `json:"appName"`
	AppVersion                         string         `json:"appVersion"`
	Duration                           int64          `json:"duration"`
	JREVersion                         string         `json:"jreVersion"`
	JREAvailableProcessors             int64          `json:"jreAvailableProcessors"`
	JREFreeMemory                      int64          `json:"jreFreeMemory"`
	JREMaxMemory                       int64          `json:"jreMaxMemory"`
	JRETotalMemory                     int64          `json:"jreTotalMemory"`
	LatestMilestone                    trinary.Trytes `json:"latestMilestone"`
	LatestMilestoneIndex               int64          `json:"latestMilestoneIndex"`
	LatestSolidSubtangleMilestone      trinary.Trytes `json:"latestSolidSubtangleMilestone"`
	LatestSolidSubtangleMilestoneIndex int64          `json:"latestSolidSubtangleMilestoneIndex"`
	Neighbors                          int64          `json:"neighbors"`
	PacketQueueSize                    int64          `json:"packetQueueSize"`
	Time                               int64          `json:"time"`
	Tips                               int64          `json:"tips"`
	TransactionsToRequest              int64          `json:"transactionsToRequest"`
}

// GetNodeInfo calls GetNodeInfo API.
func (api *API) GetNodeInfo() (*GetNodeInfoResponse, error) {
	resp := &GetNodeInfoResponse{}
	err := api.do(map[string]string{
		"command": "getNodeInfo",
	}, resp)

	return resp, err
}

// CheckConsistency calls CheckConsistency API which returns true if confirming
// the specified tails would result in a consistent ledger state.
func (api *API) CheckConsistency(tails []trinary.Trytes) (*CheckConsistencyResponse, error) {
	resp := &CheckConsistencyResponse{}
	err := api.do(&struct {
		Command string           `json:"command"`
		Tails   []trinary.Trytes `json:"tails"`
	}{
		"checkConsistency",
		tails,
	}, resp)

	return resp, err
}

// CheckConsistencyResponse is for CheckConsistency API response.
type CheckConsistencyResponse struct {
	Duration int64  `json:"duration"`
	State    bool   `json:"state"`
	Info     string `json:"info"`
}

// Neighbor is a part of response of GetNeighbors API.
type Neighbor struct {
	Address                           signing.Address `json:"address"`
	ConnectionType                    string          `json:"connectionType"`
	NumberOfAllTransactions           int64           `json:"numberOfAllTransactions"`
	NumberOfInvalidTransactions       int64           `json:"numberOfInvalidTransactions"`
	NumberOfNewTransactions           int64           `json:"numberOfNewTransactions"`
	NumberOfRandomTransactionRequests int64           `json:"numberOfRandomTransactionRequests"`
	NumberOfSentTransactions          int64           `json:"numberOfSentTransactions"`
}

// GetNeighborsRequest is for GetNeighbors API request.
type GetNeighborsRequest struct {
	Command string `json:"command"`
}

// GetNeighborsResponse is for GetNeighbors API response.
type GetNeighborsResponse struct {
	Duration  int64
	Neighbors []Neighbor
}

// GetNeighbors calls GetNeighbors API.
func (api *API) GetNeighbors() (*GetNeighborsResponse, error) {
	resp := &GetNeighborsResponse{}
	err := api.do(map[string]string{
		"command": "getNeighbors",
	}, resp)

	return resp, err
}

// AddNeighborsRequest is for AddNeighbors API request.
type AddNeighborsRequest struct {
	Command string `json:"command"`

	// URIS is an array of strings in the form of "udp://identifier:port"
	// where identifier can be either an IP address or a domain name.
	URIS []string `json:"uris"`
}

// AddNeighborsResponse is for AddNeighbors API response.
type AddNeighborsResponse struct {
	Duration       int64 `json:"duration"`
	AddedNeighbors int64 `json:"addedNeighbors"`
}

// AddNeighbors calls AddNeighbors API.
func (api *API) AddNeighbors(uris []string) (*AddNeighborsResponse, error) {
	resp := &AddNeighborsResponse{}
	err := api.do(&struct {
		Command string   `json:"command"`
		URIS    []string `json:"uris"`
	}{
		"addNeighbors",
		uris,
	}, resp)

	return resp, err
}

// RemoveNeighborsRequest is for RemoveNeighbors API request.
type RemoveNeighborsRequest struct {
	Command string `json:"command"`

	// URIS is an array of strings in the form of "udp://identifier:port"
	// where identifier can be either an IP address or a domain name.
	URIS []string `json:"uris"`
}

// RemoveNeighborsResponse is for RemoveNeighbors API response.
type RemoveNeighborsResponse struct {
	Duration         int64 `json:"duration"`
	RemovedNeighbors int64 `json:"removedNeighbors"`
}

// RemoveNeighbors calls RemoveNeighbors API.
func (api *API) RemoveNeighbors(uris []string) (*RemoveNeighborsResponse, error) {
	resp := &RemoveNeighborsResponse{}
	err := api.do(&struct {
		Command string   `json:"command"`
		URIS    []string `json:"uris"`
	}{
		"removeNeighbors",
		uris,
	}, resp)

	return resp, err
}

// GetTipsRequest is for GetTipsRequest API request.
type GetTipsRequest struct {
	Command string `json:"command"`
}

// GetTipsResponse is for GetTips API response.
type GetTipsResponse struct {
	Duration int64            `json:"duration"`
	Hashes   []trinary.Trytes `json:"hashes"`
}

// GetTips calls GetTips API.
func (api *API) GetTips() (*GetTipsResponse, error) {
	resp := &GetTipsResponse{}
	err := api.do(map[string]string{
		"command": "getTips",
	}, resp)

	return resp, err
}

// FindTransactionsRequest is for FindTransactions API request.
type FindTransactionsRequest struct {
	Command   string            `json:"command"`
	Bundles   []trinary.Trytes  `json:"bundles,omitempty"`
	Addresses []signing.Address `json:"addresses,omitempty"`
	Tags      []trinary.Trytes  `json:"tags,omitempty"`
	Approvees []trinary.Trytes  `json:"approvees,omitempty"`
}

// FindTransactionsResponse is for FindTransaction API response.
type FindTransactionsResponse struct {
	Duration int64            `json:"duration"`
	Hashes   []trinary.Trytes `json:"hashes"`
}

// FindTransactions calls FindTransactions API.
func (api *API) FindTransactions(ft *FindTransactionsRequest) (*FindTransactionsResponse, error) {
	resp := &FindTransactionsResponse{}
	err := api.do(&struct {
		Command string `json:"command"`
		*FindTransactionsRequest
	}{
		"findTransactions",
		ft,
	}, resp)

	return resp, err
}

// GetTrytesRequest is for GetTrytes API request.
type GetTrytesRequest struct {
	Command string           `json:"command"`
	Hashes  []trinary.Trytes `json:"hashes"`
}

// GetTrytesResponse is for GetTrytes API response.
type GetTrytesResponse struct {
	Duration int64                     `json:"duration"`
	Trytes   []transaction.Transaction `json:"trytes"`
}

// GetTrytes calls GetTrytes API.
func (api *API) GetTrytes(hashes ...trinary.Trytes) (*GetTrytesResponse, error) {
	resp := &GetTrytesResponse{}
	err := api.do(&struct {
		Command string           `json:"command"`
		Hashes  []trinary.Trytes `json:"hashes"`
	}{
		"getTrytes",
		hashes,
	}, resp)

	return resp, err
}

// GetInclusionStatesRequest is for GetInclusionStates API request.
type GetInclusionStatesRequest struct {
	Command      string           `json:"command"`
	Transactions []trinary.Trytes `json:"transactions"`
	Tips         []trinary.Trytes `json:"tips"`
}

// GetInclusionStatesResponse is for GetInclusionStates API response.
type GetInclusionStatesResponse struct {
	Duration int64  `json:"duration"`
	States   []bool `json:"states"`
}

// GetInclusionStates calls GetInclusionStates API.
func (api *API) GetInclusionStates(tx []trinary.Trytes, tips []trinary.Trytes) (*GetInclusionStatesResponse, error) {
	resp := &GetInclusionStatesResponse{}
	err := api.do(&struct {
		Command      string           `json:"command"`
		Transactions []trinary.Trytes `json:"transactions"`
		Tips         []trinary.Trytes `json:"tips"`
	}{
		"getInclusionStates",
		tx,
		tips,
	}, resp)

	return resp, err
}

// Balance is the total balance of an Address.
type Balance struct {
	Address signing.Address
	Value   int64
	Index   int
}

// Balances is a slice of Balance.
type Balances []Balance

// Total returns the total balance.
func (bs Balances) Total() int64 {
	var total int64
	for _, b := range bs {
		total += b.Value
	}
	return total
}

// GetBalancesRequest is for GetBalances API request.
type GetBalancesRequest struct {
	Command   string            `json:"command"`
	Addresses []signing.Address `json:"addresses"`
	Threshold int64             `json:"threshold"`
}

// GetBalancesResponse is for GetBalances API response.
type GetBalancesResponse struct {
	Duration       int64          `json:"duration"`
	Balances       []int64        `json:"balances"`
	Milestone      trinary.Trytes `json:"milestone"`
	MilestoneIndex int64          `json:"milestoneIndex"`
}

// Balances call GetBalances API and returns address-balance pair struct.
func (api *API) Balances(addrs []signing.Address) (Balances, error) {
	r, err := api.GetBalances(addrs, 100)
	if err != nil {
		return nil, err
	}

	bs := make(Balances, 0, len(addrs))
	for i, bal := range r.Balances {
		if bal <= 0 {
			continue
		}
		b := Balance{
			Address: addrs[i],
			Value:   bal,
			Index:   i,
		}
		bs = append(bs, b)
	}
	return bs, nil
}

// GetBalances calls GetBalances API.
func (api *API) GetBalances(adr []signing.Address, threshold int64) (*GetBalancesResponse, error) {
	if threshold <= 0 {
		threshold = 100
	}

	type getBalancesResponse struct {
		Duration       int64          `json:"duration"`
		Balances       []string       `json:"balances"`
		Milestone      trinary.Trytes `json:"milestone"`
		MilestoneIndex int64          `json:"milestoneIndex"`
	}

	resp := &getBalancesResponse{}
	err := api.do(&struct {
		Command   string            `json:"command"`
		Addresses []signing.Address `json:"addresses"`
		Threshold int64             `json:"threshold"`
	}{
		"getBalances",
		adr,
		threshold,
	}, resp)

	r := &GetBalancesResponse{
		Duration:       resp.Duration,
		Balances:       make([]int64, len(resp.Balances)),
		Milestone:      resp.Milestone,
		MilestoneIndex: resp.MilestoneIndex,
	}

	for i, ba := range resp.Balances {
		r.Balances[i], err = strconv.ParseInt(ba, 10, 64)
		if err != nil {
			return nil, err
		}
	}
	return r, err
}

// GetTransactionsToApproveRequest is for GetTransactionsToApprove API request.
type GetTransactionsToApproveRequest struct {
	Command string `json:"command"`
	Depth   int64  `json:"depth"`
}

// GetTransactionsToApproveResponse is for GetTransactionsToApprove API response.
type GetTransactionsToApproveResponse struct {
	Duration          int64          `json:"duration"`
	TrunkTransaction  trinary.Trytes `json:"trunkTransaction"`
	BranchTransaction trinary.Trytes `json:"branchTransaction"`
}

// GetTransactionsToApprove calls GetTransactionsToApprove API.
func (api *API) GetTransactionsToApprove(depth int, reference trinary.Trytes) (*GetTransactionsToApproveResponse, error) {
	resp := &GetTransactionsToApproveResponse{}
	err := api.do(&struct {
		Command   string         `json:"command"`
		Depth     int            `json:"depth"`
		Reference trinary.Trytes `json:"reference,omitempty"`
	}{
		"getTransactionsToApprove",
		depth,
		reference,
	}, resp)

	return resp, err
}

// AttachToTangleRequest is for AttachToTangle API request.
type AttachToTangleRequest struct {
	Command            string                    `json:"command"`
	TrunkTransaction   trinary.Trytes            `json:"trunkTransaction"`
	BranchTransaction  trinary.Trytes            `json:"branchTransaction"`
	MinWeightMagnitude int64                     `json:"minWeightMagnitude"`
	Trytes             []transaction.Transaction `json:"trytes"`
}

// AttachToTangleResponse is for AttachToTangle API response.
type AttachToTangleResponse struct {
	Duration int64                     `json:"duration"`
	Trytes   []transaction.Transaction `json:"trytes"`
}

// AttachToTangle calls AttachToTangle API.
func (api *API) AttachToTangle(att *AttachToTangleRequest) (*AttachToTangleResponse, error) {
	resp := &AttachToTangleResponse{}
	err := api.do(&struct {
		Command string `json:"command"`
		*AttachToTangleRequest
	}{
		"attachToTangle",
		att,
	}, resp)

	return resp, err
}

// InterruptAttachingToTangleRequest is for InterruptAttachingToTangle API request.
type InterruptAttachingToTangleRequest struct {
	Command string `json:"command"`
}

// InterruptAttachingToTangle calls InterruptAttachingToTangle API.
func (api *API) InterruptAttachingToTangle() error {
	err := api.do(map[string]string{
		"command": "interruptAttachingToTangle",
	}, nil)

	return err
}

type AccountData struct {
	LatestAddress signing.Address    `json:"latest_address"`
	Transfers     []bundle.Transfers `json:"transfers"`
	Transactions  []trinary.Trytes   `json:"transactions"`
	Inputs        []trinary.Trytes   `json:"inputs"`
	Addresses     []trinary.Trytes   `json:"addresses"`
}

func (api *API) AccountData(seed trinary.Trytes, startIndex int, endIndex int, securityLvl int) (*AccountData, error) {

	// 1. generate addresses up to first unused address

	// 2. query to fetch complete bundles, balances and spending states of addresses

	return nil, nil
}

// BroadcastBundle re-broadcasts all transactions in a bundle given the tail transaction hash.
// It might be useful when transactions did not properly propagate, particularly in the case of large bundles.
func (api *API) BroadcastBundle(tailTransactionHash trinary.Trytes) error {
	var getTrytesRes *GetTrytesResponse
	var err error

	getTrytesRes, err = api.GetTrytes(tailTransactionHash)
	if err != nil {
		return err
	}

	tx := getTrytesRes.Trytes[0]

	// check whether we actually got the tail transaction passed in
	if tx.CurrentIndex != 0 {
		return ErrInvalidTailTransactionHash
	}

	txsInBundle := int(tx.LastIndex + 1)
	bundle := make(bundle.Bundle, txsInBundle)
	bundle[0] = tx

	for i := 1; i < txsInBundle; i++ {
		getTrytesRes, err = api.GetTrytes(tx.TrunkTransaction)
		if err != nil {
			return err
		}

		tx = getTrytesRes.Trytes[0]
		bundle[i] = tx
	}

	return api.BroadcastTransactions(bundle)
}

// BroadcastTransactionsRequest is for BroadcastTransactions API request.
type BroadcastTransactionsRequest struct {
	Command string                    `json:"command"`
	Trytes  []transaction.Transaction `json:"trytes"`
}

// BroadcastTransactions calls BroadcastTransactions API.
func (api *API) BroadcastTransactions(trytes []transaction.Transaction) error {
	err := api.do(&struct {
		Command string                    `json:"command"`
		Trytes  []transaction.Transaction `json:"trytes"`
	}{
		"broadcastTransactions",
		trytes,
	}, nil)

	return err
}

// StoreTransactionsRequest is for StoreTransactions API request.
type StoreTransactionsRequest struct {
	Command string                    `json:"command"`
	Trytes  []transaction.Transaction `json:"trytes"`
}

// StoreTransactions calls StoreTransactions API.
func (api *API) StoreTransactions(trytes []transaction.Transaction) error {
	err := api.do(&struct {
		Command string                    `json:"command"`
		Trytes  []transaction.Transaction `json:"trytes"`
	}{
		"storeTransactions",
		trytes,
	}, nil)

	return err
}

// GetLatestInclusion takes the most recent solid milestone as returned by getNodeInfo
// and uses it to get the inclusion states of a list of transaction hashes
func (api *API) GetLatestInclusion(hash []trinary.Trytes) ([]bool, error) {
	var (
		gt   *GetTrytesResponse
		ni   *GetNodeInfoResponse
		err1 error
		err2 error
	)

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		gt, err1 = api.GetTrytes(hash...)
		wg.Done()
	}()

	go func() {
		ni, err2 = api.GetNodeInfo()
		wg.Done()
	}()

	wg.Wait()

	switch {
	case err1 != nil:
		return nil, err1
	case err2 != nil:
		return nil, err2
	case len(gt.Trytes) == 0:
		return nil, ErrTxNotFound
	}

	resp, err := api.GetInclusionStates(hash, []trinary.Trytes{ni.LatestMilestone})
	if err != nil {
		return nil, err
	}

	if len(resp.States) == 0 {
		return nil, ErrTxNotFoundInInclusionState
	}
	return resp.States, nil
}

// GetUsedAddress generates a new address which is not found in the tangle
// and returns its new address and used addresses.
func (api *API) GetUsedAddress(seed trinary.Trytes, security int) (signing.Address, []signing.Address, error) {
	var all []signing.Address
	for index := 0; ; index++ {
		adr, err := signing.NewAddress(seed, index, security)
		if err != nil {
			return "", nil, err
		}

		r := FindTransactionsRequest{
			Addresses: []signing.Address{adr},
		}

		resp, err := api.FindTransactions(&r)
		if err != nil {
			return "", nil, err
		}

		if len(resp.Hashes) == 0 {
			return adr, all, nil
		}

		// reached the end of the loop, so must be used address, repeat until return
		all = append(all, adr)
	}
}

// GetInputs gets all possible inputs of a seed and returns them with the total balance.
// end must be under start+500.
func (api *API) GetInputs(seed trinary.Trytes, start, end int, threshold int64, security int) (Balances, error) {
	var err error
	var adrs []signing.Address

	if start > end || end > (start+500) {
		return nil, errors.New("Invalid start/end provided")
	}

	switch {
	case end > 0:
		adrs, err = signing.NewAddresses(seed, start, end-start, security)
	default:
		_, adrs, err = api.GetUsedAddress(seed, security)
	}

	if err != nil {
		return nil, err
	}

	return api.Balances(adrs)
}

// gets all balances of the given inputs or if none supplied,
// deterministically computes the balance from up to 100 addresses.
// the supplied total must be less than the actual computed balance,
// otherwise an error is returned
func (api *API) setupInputs(seed trinary.Trytes, inputs bundle.AddressInfos, security int, total int64) (Balances, bundle.AddressInfos, error) {
	var balances Balances
	var err error

	if inputs != nil {
		// gather all addresses and balances from the provided address infos
		addrs := make([]signing.Address, len(inputs))
		for i, ai := range inputs {
			addrs[i], err = ai.Address()
			if err != nil {
				return nil, nil, err
			}
		}

		//  Validate the inputs by calling getBalances (in call to Balances)
		balances, err = api.Balances(addrs)
		if err != nil {
			return nil, nil, err
		}
	} else {
		// If inputs with enough balance
		balances, err = api.GetInputs(seed, 0, 100, total, security)
		if err != nil {
			return nil, nil, err
		}

		inputs = make(bundle.AddressInfos, len(balances))
		for i := range balances {
			inputs[i].Index = balances[i].Index
			inputs[i].Security = security
			inputs[i].Seed = seed
		}
	}

	// Return not enough balance error
	if total > balances.Total() {
		return nil, nil, errors.New("Not enough balance")
	}
	return balances, inputs, nil
}

// PrepareTransfers gets an array of transfer objects as input, and then prepares
// the transfer by generating the correct bundle as well as choosing and signing the
// inputs if necessary (if it's a value transfer).
func (api *API) PrepareTransfers(seed trinary.Trytes, transfers bundle.Transfers, inputs bundle.AddressInfos, remainder signing.Address, security int) (bundle.Bundle, error) {
	var err error

	bundle, frags, total := transfers.AddOutputs()

	// Get inputs if we are sending tokens
	if total <= 0 {
		// If no input required, don't sign and simply finalize the bundle
		bundle.Finalize(frags)
		return bundle, nil
	}

	balances, inputs, err := api.setupInputs(seed, inputs, security, total)
	if err != nil {
		return nil, err
	}

	err = api.AddRemainder(balances, &bundle, security, remainder, seed, total)
	if err != nil {
		return nil, err
	}

	bundle.Finalize(frags)
	err = bundle.SignInputs(inputs)
	return bundle, err
}

func (api *API) AddRemainder(in Balances, bundle *bundle.Bundle, security int, remainder signing.Address, seed trinary.Trytes, total int64) error {
	for _, bal := range in {
		var err error

		// Add input as bundle entry
		bundle.Add(security, bal.Address, -bal.Value, time.Now(), curl.EmptyHash)

		// If there is a remainder value add extra output to send remaining funds to
		if remain := bal.Value - total; remain > 0 {
			// If user has provided remainder address use it to send remaining funds to
			adr := remainder
			if adr == "" {
				// Generate a new Address by calling getNewAddress
				adr, _, err = api.GetUsedAddress(seed, security)
				if err != nil {
					return err
				}
			}

			// Remainder bundle entry
			bundle.Add(1, adr, remain, time.Now(), curl.EmptyHash)
			return nil
		}

		// If multiple inputs provided, subtract the totalTransferValue by
		// the inputs balance
		if total -= bal.Value; total == 0 {
			return nil
		}
	}
	return nil
}

// SendTrytes does attachToTangle and finally, it broadcasts and stores the transactions.
func (api *API) SendTrytes(depth int, trytes []transaction.Transaction, mwm int64, pow pow.PowFunc) error {
	tra, err := api.GetTransactionsToApprove(depth, "")
	if err != nil {
		return err
	}

	switch {
	case pow == nil:
		at := AttachToTangleRequest{
			TrunkTransaction:   tra.TrunkTransaction,
			BranchTransaction:  tra.BranchTransaction,
			MinWeightMagnitude: mwm,
			Trytes:             trytes,
		}

		// attach to tangle - do pow
		attached, err := api.AttachToTangle(&at)
		if err != nil {
			return err
		}

		trytes = attached.Trytes
	default:
		err := bundle.DoPoW(tra.TrunkTransaction, tra.BranchTransaction, trytes, mwm, pow)
		if err != nil {
			return err
		}
	}

	// Broadcast and store tx
	err = api.BroadcastTransactions(trytes)
	if err != nil {
		return err
	}

	return api.StoreTransactions(trytes)
}

// Promote sends a transaction using tail as reference (promotes the tail transaction)
func (api *API) Promote(tail trinary.Trytes, depth int, trytes []transaction.Transaction, mwm int64, pow pow.PowFunc) error {
	if len(trytes) == 0 {
		return errors.New("empty transfer")
	}
	resp, err := api.CheckConsistency([]trinary.Trytes{tail})
	if err != nil {
		return err
	} else if !resp.State {
		return errors.New(resp.Info)
	}

	tra, err := api.GetTransactionsToApprove(depth, tail)
	if err != nil {
		return err
	}

	switch {
	case pow == nil:
		at := AttachToTangleRequest{
			TrunkTransaction:   tra.TrunkTransaction,
			BranchTransaction:  tra.BranchTransaction,
			MinWeightMagnitude: mwm,
			Trytes:             trytes,
		}

		// attach to tangle - do pow
		attached, err := api.AttachToTangle(&at)
		if err != nil {
			return err
		}

		trytes = attached.Trytes
	default:
		err := bundle.DoPoW(tra.TrunkTransaction, tra.BranchTransaction, trytes, mwm, pow)
		if err != nil {
			return err
		}
	}

	// Broadcast and store tx
	err = api.BroadcastTransactions(trytes)
	if err != nil {
		return err
	}

	return api.StoreTransactions(trytes)
}

// Send sends tokens. If you need to do pow locally, you must specify pow func,
// otherwise this calls the AttachToTangle API
func (api *API) Send(seed trinary.Trytes, security int, depth int, transfers bundle.Transfers, mwm int64, pow pow.PowFunc) (bundle.Bundle, error) {
	bd, err := api.PrepareTransfers(seed, transfers, nil, "", security)
	if err != nil {
		return nil, err
	}

	err = api.SendTrytes(depth, []transaction.Transaction(bd), mwm, pow)
	return bd, err
}
