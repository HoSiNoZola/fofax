package iconhash

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"hash"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/twmb/murmur3"
	"github.com/xiecat/fofax/internal/printer"
)

type Config struct {
	Debug         bool
	FoFaFormat    bool
	ShoDanFormat  bool
	CenSysFormat  bool
	ZoomEyeFormat bool
	IsUint32      bool
	HashUrl       string
	HashFilePath  string
	SkipVerify    bool
	UserAgent     string
	client        *http.Client
}

var config Config

func NewIconHashConfig(xclient *http.Client, url string, debug bool) *Config {
	return &Config{
		client:       xclient,
		Debug:        debug,
		HashUrl:      url,
		HashFilePath: "",
		SkipVerify:   true,
		UserAgent:    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_0) AppleWebKit/535.11 (KHTML, like Gecko) Chrome/17.0.963.56 Safari/535.11",
	}
}

func (c *Config) MakeQuery(result string) string {
	if c.FoFaFormat {
		return fmt.Sprintf("icon_hash=\"%s\"", result)
	}

	if c.ShoDanFormat {
		return fmt.Sprintf("http.favicon.hash:%s", result)
	}
	return fmt.Sprintf("icon_hash=\"%s\"", result)
}

func (c *Config) FromUrlGetContent() (string, error) {
	if c.Debug {
		printer.Info("---------------------------  start url  content  --------------------------------\n")
		printer.Infof("====> url: %s\n", c.HashUrl)
		defer printer.Info("---------------------------  end url  content  --------------------------------\n")
	}

	req, err := http.NewRequest("GET", c.HashUrl, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if c.Debug {
		printer.Infof("===> status code: %d\n", resp.StatusCode)
		printer.Infof("====> content: \n%s\n", string(body))
	}

	resp.Body.Close()
	if err != nil {
		return "", err
	}
	return Mmh3Hash32(StandBase64(body)), nil
}

func (c *Config) FromFileGetContent() (string, error) {

	if c.Debug {
		printer.Info("---------------------------start From file get content--------------------------------\n")
		printer.Infof("file path: %s", c.HashFilePath)
		defer printer.Info("---------------------------end  From file get content--------------------------------\n")
	}
	fi, err := os.Open(c.HashFilePath)
	if err != nil {
		return "", err
	}
	defer fi.Close()
	content, err := ioutil.ReadAll(fi)
	if c.Debug {
		printer.Debugf("====> fileContent:\n %s\n", content)
	}
	if err != nil {
		return "", err
	}
	return Mmh3Hash32(StandBase64(content)), nil
}

func Mmh3Hash32(raw []byte) string {
	var h32 hash.Hash32 = murmur3.New32()
	h32.Write([]byte(raw))
	if config.IsUint32 {
		return fmt.Sprintf("%d", h32.Sum32())
	}
	return fmt.Sprintf("%d", int32(h32.Sum32()))
}

func StandBase64(braw []byte) []byte {
	bckd := base64.StdEncoding.EncodeToString(braw)
	var buffer bytes.Buffer
	for i := 0; i < len(bckd); i++ {
		ch := bckd[i]
		buffer.WriteByte(ch)
		if (i+1)%76 == 0 {
			buffer.WriteByte('\n')
		}
	}
	buffer.WriteByte('\n')
	if config.Debug {
		printer.Debugf("---------------------------start base64 content--------------------------------\n")
		printer.Debugf("====> base64:\n%s\n", buffer.String())
		defer printer.Debugf("---------------------------end base64 content--------------------------------\n")
	}
	return buffer.Bytes()

}

func (c *Config) SplitChar76(braw []byte) []byte {
	// 去掉 data:image/vnd.microsoft.icon;base64
	if strings.HasPrefix(string(braw), "data:image/vnd.microsoft.icon;base64,") {
		braw = braw[37:]
	}

	var buffer bytes.Buffer
	for i := 0; i < len(braw); i++ {
		ch := braw[i]
		buffer.WriteByte(ch)
		if (i+1)%76 == 0 {
			buffer.WriteByte('\n')
		}
	}
	buffer.WriteByte('\n')

	if c.Debug {
		printer.Debugf("---------------------------start base64 content--------------------------------\n")
		printer.Debugf("====> base64 split 76:\n %s\n", buffer.String())
		printer.Debugf("---------------------------end base64 content--------------------------------\n")
	}

	return buffer.Bytes()

}
