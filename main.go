package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/pkg/errors"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg/draw"
)

func main() {
	rand.Seed(time.Now().Unix())

	var s server

	http.HandleFunc("/", s.root)
	http.HandleFunc("/statz", s.statz)
	http.HandleFunc("/statz/scatter.png", errorHandler(s.scatter))
	http.HandleFunc("/statz/hist.png", errorHandler(s.hist))
	log.Fatal(http.ListenAndServe("localhost:8080", nil))

}

type server struct {
	data []time.Duration
	sync.RWMutex
}

func (s *server) root(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	//	x := 1000 * rand.Float64()
	x := 500 + 200*rand.NormFloat64()
	d := time.Duration(x) * time.Millisecond
	// don't sleep - generate LOTS of data quickly
	//time.Sleep(d)
	fmt.Fprintln(w, "slept for : ", d)

	s.Lock()

	s.data = append(s.data, d)
	if len(s.data) > 1000 {
		s.data = s.data[len(s.data)-1000:]
	}

	s.Unlock()

}

func (s *server) statz(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%s", `
    <html>
    <h1>Latency Stats</h1>
    <img src="/statz/scatter.png?rand=0" style="width:40%;">
    <img src="/statz/hist.png?rand=0" style="width:40%;">
    <script>
    setInterval(function() {
      var imgs = document.getElementsByTagName("IMG");
      for (var i=0; i < imgs.length; i++) {
        var eqPos = imgs[i].src.lastIndexOf("=");
        var src = imgs[i].src.substr(0, eqPos+1);
        imgs[i].src = src + Math.random();
      }
      }, 1000);
    </script>
    </html>
    `)
}

func (s *server) scatter(w http.ResponseWriter, r *http.Request) error {
	s.RLock()
	defer s.RUnlock()

	xys := make(plotter.XYs, len(s.data))
	for i, d := range s.data {
		xys[i].X = float64(i)
		xys[i].Y = float64(d) / float64(time.Millisecond)
	}
	sc, err := plotter.NewScatter(xys)
	if err != nil {
		return errors.Wrap(err, "could not create scatter")
	}
	sc.GlyphStyle.Shape = draw.CrossGlyph{}

	avgs := make(plotter.XYs, len(s.data))
	sum := 0.0
	for i, d := range s.data {
		avgs[i].X = float64(i)
		sum += float64(d)
		avgs[i].Y = sum / (float64(time.Millisecond) * float64(i+1))
	}
	l, err := plotter.NewLine(avgs)
	if err != nil {
		return errors.Wrap(err, "could not create line")
	}
	l.Color = color.RGBA{G: 255, A: 255}

	g := plotter.NewGrid()
	g.Horizontal.Color = color.RGBA{R: 255, A: 255}
	g.Vertical.Width = 0

	p, err := plot.New()
	if err != nil {
		return errors.Wrap(err, "could not create plot")
	}

	p.Add(sc, l, g)
	p.Title.Text = "Endpoint Latency"
	p.Y.Label.Text = "ms"
	p.X.Label.Text = "sample"

	wt, err := p.WriterTo(512, 512, "png")
	if err != nil {
		return errors.Wrap(err, "could not create WriterTo")
	}

	// w.Header().Set("Content-Type", "image/png")
	_, err = wt.WriteTo(w)
	return errors.Wrap(err, "could not write to output")
	/*
		for _, d := range s.data {
			fmt.Fprintln(w, d)
		} */
}

func errorHandler(h func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (s *server) hist(w http.ResponseWriter, r *http.Request) error {
	s.RLock()
	defer s.RUnlock()

	vs := make(plotter.Values, len(s.data))
	for i, d := range s.data {
		vs[i] = float64(d) / float64(time.Millisecond)
	}

	h, err := plotter.NewHist(vs, 50)
	if err != nil {
		return errors.Wrap(err, "could not create histogram")
	}

	p, err := plot.New()
	if err != nil {
		return errors.Wrap(err, "could not create plot")
	}

	p.Add(h)
	p.Title.Text = "Distribution"
	p.X.Label.Text = "ms"

	wt, err := p.WriterTo(512, 512, "png")
	if err != nil {
		return errors.Wrap(err, "could not create WriterTo")
	}

	// w.Header().Set("Content-Type", "image/png")
	_, err = wt.WriteTo(w)
	return errors.Wrap(err, "could not write to output")
	/*
		for _, d := range s.data {
			fmt.Fprintln(w, d)
		} */
}
