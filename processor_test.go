package workflow

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/percybolmer/workflow/handlers/files"
	"github.com/percybolmer/workflow/handlers/filters"
	"github.com/percybolmer/workflow/handlers/parsers"
	"github.com/percybolmer/workflow/handlers/terminal"
	"github.com/percybolmer/workflow/payload"
	"github.com/percybolmer/workflow/pubsub"
)

func TestNewProcessor(t *testing.T) {

	p1 := NewProcessor("test")

	if p1.ID == 0 {
		t.Fatal("should not be 0")
	}

	p2 := NewProcessor("test")
	if p2.ID == p1.ID {
		t.Fatal("Second ID should be not be duplicate ")
	}
}

func TestStart(t *testing.T) {
	// Register Printer
	// Reset Register
	testf := terminal.NewStdoutHandler()
	cfg := testf.GetConfiguration()
	// Add Properties here to test failure of them
	cfg.AddProperty("test", "testing", true)
	p := NewProcessor("test")

	err := p.Start(nil)
	if !errors.Is(err, ErrProcessorHasNoHandlerApplied) {
		t.Fatal(err)
	}
	p.SetHandler(testf)
	err = p.Start(nil)
	if !errors.Is(err, ErrNilContext) {
		t.Fatal(err)
	}
	// Invalid Properties
	err = p.Start(context.Background())
	if !errors.Is(err, ErrRequiredPropertiesNotFulfilled) {
		t.Fatal(err)
	}

	cfg.SetProperty("test", true)
	err = p.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}

func TestPubSub(t *testing.T) {

	testf := terminal.NewStdoutHandler()
	printer := NewProcessor("testf")
	printer.SetHandler(testf)
	printer.Subscribe("testtopic")
	err := printer.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Now everything is setup, What sender sends will be printed by Printer
	pay := payload.BasePayload{
		Source:  "Test",
		Payload: []byte(`Hello world`),
	}
	errs := pubsub.Publish("testtopic", pay)
	if errs != nil {
		t.Fatal(errs[0])
	}
	pubsub.Publish("topicthatbuffers", pay)
	time.Sleep(1 * time.Second)
	metricName := fmt.Sprintf("%s_%d_payloads_in", printer.Name, printer.ID)
	printerMetric := printer.Metric.GetMetrics()
	t.Logf("%+v", printerMetric)
	if printerMetric[metricName].Value != 1 {
		t.Fatal("Wrong Printer metric value")
	}

	printer2 := NewProcessor("bufferProcessor")
	printer2.SetHandler(testf)
	printer2.Subscribe("topicthatbuffers")

	printer2.Start(context.Background())
	if len(pubsub.Topics["topicthatbuffers"].Buffer.Flow) != 1 {
		t.Fatal("Wrong buffer length in topicthatbuffers")
	}
	pubsub.DrainTopicsBuffer()
	time.Sleep(1 * time.Second)
	metricName = fmt.Sprintf("%s_%d_payloads_in", printer2.Name, printer2.ID)
	printerMetric = printer2.Metric.GetMetrics()
	if printerMetric[metricName].Value != 1 {
		t.Fatal("Wrong Printer2 metric value")
	}

}

func TestRealLifeCase(t *testing.T) {
	// The idea here is to test a case of how it could be used by others
	listDirProc := NewProcessor("listdir", "found_files")
	readFileProc := NewProcessor("readfile", "file_data")
	writeFileProc := NewProcessor("writefile")
	csvReader := NewProcessor("csvReader", "map_reduce")
	filter := NewProcessor("filter", "print_stdout")

	printerProc := NewProcessor("printer", "printer_output")
	printer2Proc := NewProcessor("printer2")
	listDirProc.SetHandler(files.NewListDirectoryHandler())
	printerProc.SetHandler(terminal.NewStdoutHandler())
	writeFileProc.SetHandler(files.NewWriteFileHandler())
	printer2Proc.SetHandler(terminal.NewStdoutHandler())
	readFileProc.SetHandler(files.NewReadFileHandler())
	csvReader.SetHandler(parsers.NewParseCSVHandler())
	filter.SetHandler(filters.NewFilterHandler())
	// Setup configurations - still a bit clonky, but LoadConfig should be impl soon
	cfg := listDirProc.GetConfiguration()
	err := cfg.SetProperty("path", "testing")
	if err != nil {
		t.Fatal(err)
	}
	err = cfg.SetProperty("buffertime", 5)
	if err != nil {
		t.Fatal(err)
	}

	filters := make(map[string][]string, 0)
	filters["userinformation"] = append(filters["userinformation"], "username:percybolmer")
	printerProc.GetConfiguration().SetProperty("forward", true)
	readFileProc.GetConfiguration().SetProperty("remove_after", false)
	writeFileProc.GetConfiguration().SetProperty("path", "testing/realexample")
	writeFileProc.GetConfiguration().SetProperty("append", true)
	writeFileProc.GetConfiguration().SetProperty("forward", false)
	filter.GetConfiguration().SetProperty("filters", filters)
	filter.GetConfiguration().SetProperty("strict", []string{"userinformation"})
	// Fix Relationships
	readFileProc.Subscribe("found_files")
	csvReader.Subscribe("file_data")
	filter.Subscribe("map_reduce")
	printerProc.Subscribe("print_stdout")
	printer2Proc.Subscribe("printer_output")
	writeFileProc.Subscribe("printer_output")

	// Startup
	if err := listDirProc.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := readFileProc.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := csvReader.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := filter.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := printer2Proc.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := printerProc.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := writeFileProc.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1 * time.Second)
	t.Logf("%+v", writeFileProc.Metric.GetMetric(fmt.Sprintf("%s_%d_payloads_in", writeFileProc.Name, writeFileProc.ID)))

	// Compare metrics so that they Match
}
