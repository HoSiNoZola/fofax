package fofa

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/xiecat/fofax/internal/cli"
	"github.com/xiecat/fofax/internal/printer"
	"github.com/xiecat/fofax/internal/utils"
)

type FoFa struct {
	page    int64
	FetchFn fieldFn
	option  *cli.Options
	client  *http.Client
}

type ApiResults struct {
	Mode    string     `json:"mode"`
	Error   bool       `json:"error"`
	ErrMsg  string     `json:"errmsg"`
	Query   string     `json:"query"`
	Page    int        `json:"page"`
	Size    int        `json:"size"`
	Results [][]string `json:"results"`
}

type fieldFn func(fields []string, allSize int32) bool

//type fixUrlFn func(hostInfo *utils.FixUrl, allSize int32) bool

func NewFoFa(option *cli.Options) *FoFa {

	return &FoFa{
		option: option,
		client: option.Xclient,
	}
}

func (f *FoFa) SetFetchCallback(fn func(fields []string, allSize int32) bool) {
	f.FetchFn = fn
}

func (f *FoFa) buildQueryUrl(queryStr string) string {
	return f.option.FoFaURL + queryStr
}

func (f *FoFa) fetchByFields(fields string, queryStr string) bool {
	f.page = 1
	maxSize := f.option.FetchSize
	if maxSize < 0 {
		// max window限制
		maxSize = 10000 * 50000
	}

	for {
		// 找到最小值
		perPage := int(math.Min(float64(maxSize), 10000))
		if f.option.Debug {
			printer.Debugf("FoFa Size : %d", perPage)
			printer.Debugf("FoFa input Query of: %s", queryStr)
		}

		var isOptionsArgs string
		if f.option.Include {
			isOptionsArgs = "&fraud=true"
		}
		if f.option.OldData {
			isOptionsArgs += "&full=true"
		}
		auth := fmt.Sprintf("key=%s", f.option.FoFaKey)

		if f.option.FoFaEmail != "" {
			auth = fmt.Sprintf("email=%s&key=%s", f.option.FoFaEmail, f.option.FoFaKey)
		}

		uri := fmt.Sprintf(
			"/api/v1/search/all?%s%s&qbase64=%s&size=%d&page=%d&fields=%s",
			auth, isOptionsArgs,
			base64.StdEncoding.EncodeToString([]byte(queryStr)),
			perPage,
			f.page,
			fields,
		)

		fullURL := f.buildQueryUrl(uri)
		if f.option.Debug {
			printer.Debug(utils.HiddenUrlKey(f.option.ShowPrivacy, fullURL))
		}
		req, err := http.NewRequest("GET", fullURL, nil)

		if err != nil {
			printer.Errorf(printer.HandlerLine("request failed: " + utils.HiddenUrlKey(f.option.ShowPrivacy, err.Error())))
			return false
		}
		if f.option.FetchFields != cli.DefaultField {
			printer.Debugf("Fields : %s", strings.Join(strings.Split(f.option.FetchFields, ","), f.option.FetchFieldsSplit))
		}
		req.Header.Set("fofax-client-%s", cli.FoFaXVersion)
		// 计算时长
		start := time.Now().UnixMilli()
		// 请求
		resp, err := f.client.Do(req)
		if err != nil {
			printer.Errorf(printer.HandlerLine("request failed: " + utils.HiddenUrlKey(f.option.ShowPrivacy, err.Error())))
			return false
		}
		if resp.StatusCode != 200 {
			printer.Errorf("Http Status Code : %d", resp.StatusCode)
			return false
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			printer.Errorf(printer.HandlerLine("body read failed: " + err.Error()))
		}
		if f.option.Debug {
			printer.Debugf("Resp Time: %.2f/millis", float64(time.Now().UnixMilli()-start))
		}

		var apiResult ApiResults
		if err := json.Unmarshal(body, &apiResult); err != nil {
			printer.Errorf("Json Unmarshal Failed: %s", string(body))
			return false
		}
		if len(apiResult.ErrMsg) != 0 && apiResult.Error == true {
			printer.Errorf("FoFa Response ErrMsg: %s", apiResult.ErrMsg)
			return false
		}
		if f.option.Debug {
			printer.Debugf("Fofa Api Query: %s", apiResult.Query)
		}
		if f.option.FetchFFIWithQueryAndSize {
			printer.Successf("Fetch Data From [%s]: [%d/%d]", queryStr, len(apiResult.Results), apiResult.Size)

		} else {
			printer.Successf("Fetch Data From FoFa: [%d/%d]", len(apiResult.Results), apiResult.Size)

		}

		for _, result := range apiResult.Results {

			if !f.FetchFn(result, int32(apiResult.Size)) {
				return true
			}
			//maxSize--
			//if maxSize == 0 {
			//	return true
			//}
		}
		// 没有数据，退出
		if len(apiResult.Results) == 0 || maxSize < perPage {
			return true
		}
		maxSize -= perPage
		if maxSize <= 0 {
			return true
		}
		f.page++
		if !f.option.Coin {
			printer.Infof("Use fofa coins to get more than 10,000 data please use -coin to confirm")
			return true
		}
		printer.Infof("The fofa coin will be deducted !!!")
		time.Sleep(time.Duration(f.option.ReqIntervalTime) * time.Millisecond)
	}
}

// FetchFullHostInfo 提取完整带协议的字段
func (f *FoFa) FetchFullHostInfo(queryStr string) bool {
	return f.fetchByFields("protocol,ip,port,host,city", queryStr)
}

// FetchOneField 提取指定的字段
func (f *FoFa) FetchOneField(field, queryStr string) bool {
	return f.fetchByFields(field, queryStr)
}

// FetchField 提取指定的字段
func (f *FoFa) FetchField(field, queryStr string) bool {
	return f.fetchByFields(field, queryStr)
}

// FetchTitlesOfDomain 提取 title
func (f *FoFa) FetchTitlesOfDomain(queryStr string) bool {
	return f.fetchByFields("protocol,ip,port,host,city,title,country", queryStr)
}

// FetchJarmOfDomain 提取 title
func (f *FoFa) FetchJarmOfDomain(queryStr string) bool {
	return f.fetchByFields("protocol,ip,port,host,city,jarm,country", queryStr)
}

func (f *FoFa) Fetch(queryStr string) bool {
	return f.fetchByFields("host,port,ip,country", queryStr)
}
