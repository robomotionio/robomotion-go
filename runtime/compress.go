package runtime

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
)

func Compress(data []byte) ([]byte, error) {
	if !Props.GetBool("robomotion.compress", true) {
		return data, nil
	}

	var b bytes.Buffer

	gz := gzip.NewWriter(&b)
	if _, err := gz.Write(data); err != nil {
		return nil, err
	}

	if err := gz.Close(); err != nil {

		return nil, err
	}

	return b.Bytes(), nil
}

func Decompress(data []byte) ([]byte, error) {
	if !Props.GetBool("robomotion.compress", true) {
		return data, nil
	}

	b := bytes.NewBuffer(data)

	zr, err := gzip.NewReader(b)
	if err != nil {
		return nil, err
	}

	raw, _ := ioutil.ReadAll(zr)
	if err := zr.Close(); err != nil {
		return nil, err
	}

	return raw, nil
}
