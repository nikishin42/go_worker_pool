package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

func resultSH(str string, out chan interface{}, mu *sync.Mutex, wg *sync.WaitGroup) {

	outLeft := make(chan string)
	go func(str string, out chan string) {
		out <- DataSignerCrc32(str)
	}(str, outLeft)

	outRight := make(chan string)
	go func(str string, out chan string) {
		mu.Lock()
		Md5Data := DataSignerMd5(str)
		mu.Unlock()
		out <- DataSignerCrc32(Md5Data)
	}(str, outRight)

	res := <-outLeft + "~" + <-outRight
	out <- res
	wg.Done()
}

func SingleHash(in, out chan interface{}) {
	mu := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	for data := range in {
		wg.Add(1)
		str := strconv.Itoa(data.(int))

		go resultSH(str, out, mu, wg)
	}
	wg.Wait()
}

func resultMH(str string, out chan interface{}, wg *sync.WaitGroup) {

	var valMH valueMH
	inMH, bufMH := make(chan valueMH), make(chan valueMH)

	for i := 0; i < 6; i++ {
		go func(in, out chan valueMH) {
			data := <-in
			res := DataSignerCrc32(strconv.Itoa(data.i) + data.val)
			data.val = res
			out <- data
		}(inMH, bufMH)
		valMH.i = i
		valMH.val = str
		inMH <- valMH
	}
	go func(in chan valueMH, out chan interface{}) {
		arr := make([]string, 6)
		for i := 0; i < 6; i++ {
			data := <-in
			arr[data.i] = data.val
		}
		out <- strings.Join(arr, "")
		wg.Done()
	}(bufMH, out)

}

func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}

	for data := range in {
		wg.Add(1)
		str := data.(string)

		go resultMH(str, out, wg)
	}
	wg.Wait()
}

type valueMH struct {
	i   int
	val string
}

func CombineResults(in, out chan interface{}) {

	var arr []string
	for val := range in {
		arr = append(arr, val.(string))
	}
	sort.Strings(arr)
	out <- strings.Join(arr, "_")
}

func startWorker(oneJob job, in, out chan interface{}, waitGroup *sync.WaitGroup) {
	oneJob(in, out)
	close(out)
	waitGroup.Done()
}

func ExecutePipeline(jobs ...job) {
	var ch []chan interface{}
	waitGroup := &sync.WaitGroup{}

	for i := 0; i < 1+len(jobs); i++ {
		ch = append(ch, make(chan interface{}))
	}

	for i, _ := range jobs {
		waitGroup.Add(1)
		go startWorker(jobs[i], ch[i], ch[i+1], waitGroup)
	}
	waitGroup.Wait()
}

func main() {
	var arr []job
	inputData := []int{0, 1}
	start := func(in, out chan interface{}) {
		for _, num := range inputData {
			out <- num
		}
	}
	finish := func(in, out chan interface{}) {
		data := <-in
		myResult, ok := data.(string)
		if !ok {
			fmt.Println("cant cast in string")
		} else {
			trueResult := "29568666068035183841425683795340791879727309630931025356555_4958044192186797981418233587017209679042592862002427381542"
			fmt.Println("EXPECTED: ", trueResult)
			fmt.Println("RETURNED: ", myResult)
			fmt.Println(myResult == trueResult)
		}

	}
	arr = append(arr, start, SingleHash, MultiHash, CombineResults, finish)

	ExecutePipeline(arr...)
}
