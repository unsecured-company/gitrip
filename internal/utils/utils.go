package utils

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

func SizeToHumanReadable(size int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}
	var i int
	var floatSize float64 = float64(size)

	for i = 0; i < len(units)-1 && floatSize >= 1024; i++ {
		floatSize /= 1024
	}

	return fmt.Sprintf("%.2f %s", floatSize, units[i])
}

func UrlStrToFolderName(urlStr string) (dirName string, err error) {
	urlObj, err := url.Parse(urlStr)

	if err != nil {
		return
	}

	return UrlParsedToFolderName(urlObj)
}

func UrlParsedToFolderName(urlP *url.URL) (dirName string, err error) {
	dirName = urlP.String()
	dirName = strings.ToLower(dirName)
	dirName = strings.ReplaceAll(dirName, "/", "_")
	dirName = strings.ReplaceAll(dirName, ".", "_")
	dirName = strings.ReplaceAll(dirName, " ", "_")

	re := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	dirName = re.ReplaceAllString(dirName, "")

	const maxLength = 150

	if len(dirName) > maxLength {
		dirName = dirName[:maxLength]
	}

	return dirName, nil
}

func DecodeZlib(input []byte) (data []byte, err error) {
	r, err := zlib.NewReader(bytes.NewReader(input))

	if err != nil {
		return
	}

	defer r.Close()

	return io.ReadAll(r)
}

func NumToUnderscores(n int) string {
	s := strconv.Itoa(n)
	reversed := reverseString(s)
	var sb strings.Builder

	for i, char := range reversed {
		if i > 0 && i%3 == 0 {
			sb.WriteRune('_')
		}

		sb.WriteRune(char)
	}

	formatted := reverseString(sb.String())

	return formatted
}

func reverseString(s string) string {
	runes := []rune(s)

	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}

	return string(runes)
}

func ParseUrlOrDomain(input string) (urlP *url.URL, err error) {
	if !strings.Contains(input, "://") && !strings.HasPrefix(input, "//") {
		input = "//" + input
	}

	return url.Parse(input)
}

// GetUrls parses a URL or domain. In case of domain returns both http(s).
func GetUrls(input string) (urls []*url.URL, err error) {
	if !strings.Contains(input, "://") && !strings.HasPrefix(input, "//") {
		input = "//" + input
	}

	parsed, err := url.Parse(input)

	if err != nil {
		return
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		urlHttp := *parsed
		urlHttps := *parsed

		urlHttp.Scheme = "http"
		urlHttps.Scheme = "https"

		urls = []*url.URL{&urlHttp, &urlHttps}
	} else {
		urls = []*url.URL{parsed}
	}

	return
}

func AddUrlSuffix(urlP *url.URL, suffix string) {
	suffix = strings.Trim(suffix, "/")
	urlP.Path = strings.TrimRight(urlP.Path, "/")

	if !strings.HasSuffix(urlP.Path, suffix) {
		urlP.Path += "/" + suffix
	}
}

func GetNewSuffixedUrl(urlIn *url.URL, suffix string) (urlOut *url.URL) {
	urlCopy := *urlIn
	urlOut = &urlCopy

	suffix = strings.Trim(suffix, "/")
	urlOut.Path = strings.TrimRight(urlOut.Path, "/")

	if !strings.HasSuffix(urlOut.Path, suffix) {
		urlOut.Path += "/" + suffix
	}

	return urlOut
}
