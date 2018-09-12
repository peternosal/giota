// +build linux,darwin,windows amd64 linux,arm64

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

package pow

import (
	"github.com/iotaledger/giota/curl"
	"github.com/iotaledger/giota/trinary"
	"github.com/iotaledger/giota/tx"
	"testing"
	"time"
)

func testPowC128(t *testing.T) float64 {
	var transaction trinary.Trytes = "SISEZJUUKSTSX9KVQGXSYYLNDIBJDVRZSOFEHWJSDZLNUUNBDLHUODEGFZQTKOEXUMMQTOREUWQCSGGWRKALQDDZCQN9LBIEVKBFDCWBIDWD9DGVOJVCNUNWDDZFCIOICZZF9KIAYDCSKJWE99UPPLUQPUSWTDKTSSTJAQNYATUTXZPA9CCJRRNIRWXTAR9ECVYXC9AOHXHYVOS9LWDUOH9SDUAQBEYTMJIMUHJTGUSQTFPRLLXIDKOVZMONJHXPCD9FYLW9PN9LLPQBJRSEKVKKJB9JRTZCXSDBMJYAKDX99EGNLFZPKIADJQEIMCKRFQKIHGCJAHPL9JFJF9PHRKPCHBPN9LYQSC9TXOXAI9WBDIBNGFPLQS9BHTEVROMCAXXAXPVBAP9URJXIVZXIWWCMVDXGAFZOIRTJIMNIZEPGFMWXWOWRDUMHFRKL9LV9VJQIRZPVJSSKHXHHVZLRZYHGWQAVL9BMWKKFGZQEYJNCGROYYDIDULQVSXGVLTTZRLPSKPVIURJ9CJBTNAYCPHQTWTTKHXPABTYYCCVAZATEVED9PBJQTNOQEQQBTSATZJTVUTZPUWDYKROBROUVSPMDLUMEZWMPESEMQPSVTDZKATUTOAEVWCW9HIKKHMOQYJOUYLTFPERSKBVWARHGJNKUWGFZYF9WSTEHEQWCA9DTOTOTNDFGAEABKKBKEFLDELEOYPZTCVKOBIWA9HWTCQT9IGYVFAFAOLOJMRDZKCBYOCPGEGGZL9CGFURM9FJBLGLZJILNSFOBXLQOZWVLAZUFLGQNCAVJTBGVLZETETWGXLPSPWMMAEGORSDGPUSFRQ9AVWWZCFNKSAHIKJOMEWCCFGVYSDYNIXYYTKJTOKZUGLKNEXHWQ9HVFVJUGJJEDQACTWPSFOONTNCJRDQBSCGXVKWZIGDK9RGHKAHSTOJDJEHIAOF9MFLAZJXLUGQUAUGKQGQIXXNLAPRQNTNVDGXVZBSEFXVRR9ZQIZEWPXZFMXLJFTFKEPPAFJTMBLBWYAWJEIHUNATL9EHIJQTCCMQFHILGHGEVXKHDCNMAHDPUGBQYYBF9CRIKDVZZ9KIFELUUKPXPRIFVTZPXRBKJBRLEGUJKXZPYGXRKOAHROFXENAUAYOSQBJGMMHIDUNSYYGQSDJDKMPNBPTUWMIYZCWABYLDMTXAGWFYEXRGLOYVPNSOVYITEPCXMTMPVLBQPBNQUBITEM99KVRTPNAAWPR9RQYBLFZDVWYDJXQRGTVAFVE99KE9YSCETBIELIWPKZYFARSPVLTDKEAKLCKULZHLKOQZMVLFLF9QHT9LLS9QQODSFYUIPKSBVSKAJMVW9QUILQSKHZMAXGVHUJBMTATPIDHJVUBZWUOYNOOMEJVOUXHACUHDVKZ9ZDTSIHQOTOVUMEISMA9VZIFQTPBXXDHDLVLKZZHLYLPIE9SKOEJXAFDKICOYIOVVAEXC9VZSFSDTSHVEOSHIT9JHMBBPQTRGOREIYQSBCMHJQIXTTQWOCKMCSGBRTJRRYWPXAGELIFPG9YX9FNNYGSJXJYTHIMWSXZH9JQIYXKFXEOHOE9YNHJIDAJUGPENZHOIFEHBSCQITVFHUOESVXOJPCNTUZR9LVQCXYUW9DITEXPG9KWYMBZQQCESNFVUOBQGCRRKFHOEKTHDHUNRXADXUMCWFJMZTMHN9VWLZATB9FF9HBGLFITNNVFCQICPRSGVFAATWYJT9GUJIAHNNJBECYSWSGEJYLHJPUOYESLVIELBMSLRZJLPKDKFGAJSSWZCQDLFDEXWAPILHLNHKCRMPLQUYESAEIWWNBCEIYSOHKPILTXPAFIZ9JMKFKJHTLHRHGZQLCEVJJMJHWTUKMKOWTZWGVZGQAOAKVGXZEZBMYPVWUGYJBIFXBACZLADFFBZIXKWSZLDOCGRQAZDCFPRAZYXUMNRJ9UKUKRAVSVMCENDJABZITDQLNCXZNXCOHKLATFFXKP9FFDYSAXISISMVYPXPWYPVEAYRNAITWJSTGXRAMMZIZF9IUORREWSFUNZOXDVCMBZJAET9PVHCQTMDTVVXLXDIXFSHPXWKBZBDJAAXSDEFXPARBU9GJJABPMCD9LGQJLRIYKGQORGCDDABAIAQC9MZDQLXFSAOLNYMWCJODEEUSIHEVHQPAIFQL9ECBBVZPHYU9HDBOYXTKWOIRGHUJMVV9UKHHREDIU9CRZFUZKAMUVRIEMKEKIMAGXSMGTEJWCWWAMRPWNINTETOTRMODTORVEURRY9RTDYQIEW99999999999999999999999999999999999999999999CMRKHWD99A99999999C99999999TNFAKVBFHHMKQKKSNJRLDIYUIGOMEOADJLNS9JGKGUIHZHIUDNQMVYCA9SZCLQOEVJPUGQGWTMETLGMUQMAKHHHHTBHVWYSJSXRVBRMHVV9WUTNMNFVDWLHQGFELTKZOISREPUJXNRBIAQVQWCCKB9DEZEXS999999M9EZGRXJ9WYSZXNDZBAJZMJ9VAMUWWWANGIVFKCUNRB9GLZZKRIMEFUK9KEFZXYDGBQJIU9SQUM999999999999999999999999999999999999999999999999999999999999999999999999999999999999999"

	s := time.Now()
	nonce, err := PowC128(transaction, tx.DefaultMinWeightMagnitude)
	ti := time.Now().Sub(s)
	if err != nil {
		t.Fatal(err)
	}

	transaction = transaction[:len(transaction)-tx.NonceTrinarySize/3] + nonce
	h := curl.Hash(transaction)
	if h[len(h)-4:] != "9999" {
		t.Error("pow is illegal", h)
	}

	return float64(countC128) / 1000 / ti.Seconds()
}

func TestPowC128(t *testing.T) {
	_proc := PowProcs

	tests := []struct {
		name     string
		powProcs int
	}{
		{
			name:     "test plain PowC128 without setting PowProcs",
			powProcs: PowProcs,
		},
		{
			name:     "test with PowProcs = 1",
			powProcs: 1,
		},
		{
			name:     "test with PowProcs = 32",
			powProcs: 32,
		},
		{
			name:     "test with PowProcs = 64",
			powProcs: 64,
		},
	}

	for _, tt := range tests {
		PowProcs = tt.powProcs
		sp := testPowC128(t)
		t.Logf("%s: %d kH/sec on SEE PoW", tt.name, int(sp))
	}

	PowProcs = _proc
}
