package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
)

// generateData writes metrics in the following format to a text file:
// 		 name, type, label1 labelvalue1 , value1 value2 value3 value4 value5
// gauge and counter has only one value
func generateData() {
	f, err := os.Create(inputPath)
	if err != nil {
		log.Println(err)
		return
	}
	defer f.Close()

	for i := 0; i < item; i++ {

		mName := metric + strconv.Itoa(i)
		mType := types[rand.Intn(len(types))]
		labelSize := rand.Intn(len(labels)) + 1
		b := &strings.Builder{}
		writeNameTypeLabel(mName, mType, labelSize, b)
		switch mType {
		case gauge, counter:
			b.WriteString(strconv.Itoa(rand.Intn(valueBound)))
		case histogram:
			count := 0
			buckets := make([]int, 3, 3)
			for i := range bounds {
				n := rand.Intn(valueBound)
				buckets[i] = n
				count += n
			}
			b.WriteString(strconv.Itoa(rand.Intn(valueBound))) // sum
			b.WriteString(space)
			b.WriteString(strconv.Itoa(count)) // count
			b.WriteString(space)
			for _, val := range buckets {
				b.WriteString(strconv.Itoa(val)) // individual bucket
				b.WriteString(space)
			}
		case summary:
			b.WriteString(strconv.Itoa(rand.Intn(valueBound))) // sum
			b.WriteString(space)
			b.WriteString(strconv.Itoa(rand.Intn(valueBound))) // count
			b.WriteString(space)
			for range bounds {
				b.WriteString(fmt.Sprintf("%f", rand.Float64())) // individual quantile
				b.WriteString(space)
			}
		}
		b.WriteString("\n")
		f.WriteString(b.String())
	}
}

// writeNameTypeLabel prints the delimited metric name, metric type, and constant label values to the StringBuilder b
func writeNameTypeLabel(mName, mType string, labelSize int, b *strings.Builder) {
	b.WriteString(mName)
	b.WriteString(randomSuffix)
	b.WriteString(delimeter)
	b.WriteString(mType)
	b.WriteString(delimeter)
	for i := 0; i < labelSize; i++ {
		b.WriteString(labels[i])
		b.WriteString(space)
	}
	b.WriteString(delimeter)
}
