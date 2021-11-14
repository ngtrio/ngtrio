package main

import (
	"bufio"
	"os"
	"strconv"

	"testing"
)

func BenchmarkJsonVallinaMarshal(b *testing.B) {
	defer bufJsonVallina.Flush()

	data, _ := os.OpenFile("data", os.O_RDONLY, 0666)
	rd := bufio.NewReader(data)

	for i := 0; i < b.N; i++ {
		title, _, _ := rd.ReadLine()
		n, _, _ := rd.ReadLine()
		pageCount, _ := strconv.ParseInt(string(n), 10, 32)
		MarshalViaJsonVallina(string(title), int32(pageCount), false)
	}
}

func BenchmarkJsonSonicMarshal(b *testing.B) {
	defer bufJsonVallina.Flush()

	data, _ := os.OpenFile("data", os.O_RDONLY, 0666)
	rd := bufio.NewReader(data)

	for i := 0; i < b.N; i++ {
		title, _, _ := rd.ReadLine()
		n, _, _ := rd.ReadLine()
		pageCount, _ := strconv.ParseInt(string(n), 10, 32)
		MarshalViaJsonSonic(string(title), int32(pageCount), false)
	}
}

func BenchmarkCapnpMarshal(b *testing.B) {
	defer bufCapnp.Flush()

	data, _ := os.OpenFile("data", os.O_RDONLY, 0666)
	rd := bufio.NewReader(data)

	for i := 0; i < b.N; i++ {
		title, _, _ := rd.ReadLine()
		n, _, _ := rd.ReadLine()
		pageCount, _ := strconv.ParseInt(string(n), 10, 32)
		MarshalViaCapnp(string(title), int32(pageCount), false)
	}
}

func BenchmarkProtoMarshal(b *testing.B) {
	defer bufCapnp.Flush()

	data, _ := os.OpenFile("data", os.O_RDONLY, 0666)
	rd := bufio.NewReader(data)

	for i := 0; i < b.N; i++ {
		title, _, _ := rd.ReadLine()
		n, _, _ := rd.ReadLine()
		pageCount, _ := strconv.ParseInt(string(n), 10, 32)
		MarshalViaProto(string(title), int32(pageCount), false)
	}
}
