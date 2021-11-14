package main

import (
	booksCapnp "benchmark/capnproto"
	booksJson "benchmark/json"
	booksProto "benchmark/protobuf"
	"bufio"
	"os"
	"strconv"

	"encoding/json"

	"capnproto.org/go/capnp/v3"
	"github.com/bytedance/sonic"
	"google.golang.org/protobuf/proto"
)

var bufJsonVallina *bufio.Writer
var bufCapnp *bufio.Writer
var bufProto *bufio.Writer
var bufJsonSonic *bufio.Writer

func init() {
	filepathJsonVallina := "bytes-json-vallina"
	filepathCapnp := "bytes-capnp"
	filepathProto := "bytes-proto"
	filepathJsonSonic := "bytes-json-sonic"

	bufJsonVallina = getBufWriter(filepathJsonVallina)
	bufCapnp = getBufWriter(filepathCapnp)
	bufProto = getBufWriter(filepathProto)
	bufJsonSonic = getBufWriter(filepathJsonSonic)
}

func getBufWriter(path string) *bufio.Writer {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		panic(err)
	}
	return bufio.NewWriter(file)
}

func main() {
	data, _ := os.OpenFile("data", os.O_RDONLY, 0666)
	rd := bufio.NewReader(data)

	for i := 0; i < 1000000; i++ {
		title, _, _ := rd.ReadLine()
		n, _, _ := rd.ReadLine()
		pageCount, _ := strconv.ParseInt(string(n), 10, 32)
		MarshalViaJsonVallina(string(title), int32(pageCount), true)
		MarshalViaCapnp(string(title), int32(pageCount), true)
		MarshalViaProto(string(title), int32(pageCount), true)
		MarshalViaJsonSonic(string(title), int32(pageCount), true)
	}
}

func MarshalViaJsonVallina(title string, pageCount int32, flag bool) {
	book := &booksJson.Book{
		Title:     title,
		PageCount: pageCount,
	}
	bytes, err := json.Marshal(book)
	if err != nil {
		panic(err)
	}

	if flag {
		_, err := bufJsonVallina.Write(bytes)
		if err != nil {
			panic(err)
		}
	}
}

func MarshalViaJsonSonic(title string, pageCount int32, flag bool) {
	book := &booksJson.Book{
		Title:     title,
		PageCount: pageCount,
	}

	bytes, err := sonic.Marshal(book)
	if err != nil {
		panic(err)
	}

	if flag {
		_, err := bufJsonSonic.Write(bytes)
		if err != nil {
			panic(err)
		}
	}
}

func MarshalViaCapnp(title string, pageCount int32, flag bool) {
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		panic(err)
	}
	book, err := booksCapnp.NewRootBook(seg)
	if err != nil {
		panic(err)
	}
	book.SetTitle(title)
	book.SetPageCount(pageCount)

	bytes, err := msg.MarshalPacked()
	if err != nil {
		panic(bytes)
	}

	if flag {
		_, err := bufCapnp.Write(bytes)
		if err != nil {
			panic(err)
		}
	}
}

func MarshalViaProto(title string, pageCount int32, flag bool) {
	book := &booksProto.Books{
		Title:     title,
		PageCount: pageCount,
	}

	bytes, _ := proto.Marshal(book)

	if flag {
		_, err := bufProto.Write(bytes)
		if err != nil {
			panic(err)
		}
	}
}
