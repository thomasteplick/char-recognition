/*
Neural Network (nn) using multilayer perceptron architecture
and the backpropagation algorithm.  This is a web application that uses
the html/template package to create the HTML.
The URL is http://127.0.0.1:8080/mlpbackprop.  There are two phases of
operation:  the training phase and the testing phase.  Epochs consising of
a sequence of examples are used to train the nn.  Each example consists
of an input vector of (x,y) coordinates and a desired class output.  The nn
itself consists of an input layer of nodes, one or more hidden layers of nodes,
and an output layer of nodes.  The nodes are connected by weighted links.  The
weights are trained by back propagating the output layer errors forward to the
input layer.  The chain rule of differential calculus is used to assign credit
for the errors in the output to the weights in the hidden layers.
The output layer outputs are subtracted from the desired to obtain the error.
The user trains first and then tests.
*/

package main

import (
	"bufio"
	"fmt"
	"html/template"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
)

const (
	addr               = "127.0.0.1:8080"             // http server listen address
	fileTrainingMLP    = "templates/trainingMLP.html" // html for training MLP
	fileTestingMLP     = "templates/testingMLP.html"  // html for testing MLP
	patternTrainingMLP = "/mlpbackprop"               // http handler for training the MLP
	patternTestingMLP  = "/mlptest"                   // http handler for testing the MLP
	xlabels            = 11                           // # labels on x axis
	ylabels            = 11                           // # labels on y axis
	fileweights        = "weights.csv"                // mlp weights
	filechars          = "encoded_chars.csv"          // encoded characters
	a                  = 1.7159                       // activation function const
	b                  = 2.0 / 3.0                    // activation function const
	K1                 = b / a
	K2                 = a * a
	dataDir            = "data/" // directory for the weights
	maxClasses         = 25
	classes            = 64 // number of classes and the number of training/testing samples
	imageSize          = 81 // character image size 9 x 9
)

// Type to contain all the HTML template actions
type PlotT struct {
	Grid         []string  // plotting grid
	Status       string    // status of the plot
	Xlabel       []string  // x-axis labels
	Ylabel       []string  // y-axis labels
	HiddenLayers string    // number of hidden layers
	LayerDepth   string    // number of Nodes in hidden layers
	Classes      string    // constant number of classes = 64
	LearningRate string    // size of weight update for each iteration
	Momentum     string    // previous weight update scaling factor
	Epochs       string    // number of epochs
	TestResults  []Results // tabulated statistics of testing
	TotalCount   string    // Results tabulation
	TotalCorrect string
}

// Type to hold the minimum and maximum data values of the MSE in the Learning Curve
type Endpoints struct {
	xmin float64
	xmax float64
	ymin float64
	ymax float64
}

// graph node
type Node struct {
	y     float64 // output of this node for forward prop
	delta float64 // local gradient for backward prop
}

// graph links
type Link struct {
	wgt      float64 // weight
	wgtDelta float64 // previous weight update used in momentum
}

type Stats struct {
	correct    []int // % correct classifcation
	classCount []int // #samples in each class
}

// training examples
type Sample struct {
	encChar []int // encoded character consisting of 81 (-1,1)
	desired int   // true class of the character
	class   int   // MLP decision, the estimate
}

// Primary data structure for holding the MLP Backprop state
type MLP struct {
	plot         *PlotT   // data to be distributed in the HTML template
	Endpoints             // embedded struct
	link         [][]Link // links in the graph
	node         [][]Node // nodes in the graph
	samples      []Sample
	statistics   Stats
	mse          []float64      // mean square error in output layer per epoch used in Learning Curve
	epochs       int            // number of epochs
	learningRate float64        // learning rate parameter
	momentum     float64        // delta weight scale constant
	hiddenLayers int            // number of hidden layers
	desired      []float64      // desired output of the sample
	layerDepth   int            // hidden layer number of nodes
	char2Class   map[string]int // convert character to class
	rows         int            // #rows in grid
	columns      int            // #columns in grid
}

// test statistics that are tabulated in HTML
type Results struct {
	Class     string // int
	Correct   string // int      percent correct
	Character string // alpha-numeric character
	Count     string // int      number of training examples in the class
}

// global variables for parse and execution of the html template
var (
	tmplTrainingMLP *template.Template
	tmplTestingMLP  *template.Template
)

// calculateMSE calculates the MSE at the output layer every
func (mlp *MLP) calculateMSE(epoch int) {
	// loop over the output layer nodes
	var err float64 = 0.0
	outputLayer := mlp.hiddenLayers + 1
	for n := 0; n < len(mlp.node[outputLayer]); n++ {
		// Calculate (desired[n] - mlp.node[L][n].y)^2 and store in mlp.mse[n]
		//fmt.Printf("n = %d, desired = %f, y = %f\n", n, mlp.desired[n], mlp.node[outputLayer][n].y)
		err = float64(mlp.desired[n]) - mlp.node[outputLayer][n].y
		err2 := err * err
		mlp.mse[epoch] += err2
	}
	mlp.mse[epoch] /= float64(classes)

	// calculate min/max mse
	if mlp.mse[epoch] < mlp.ymin {
		mlp.ymin = mlp.mse[epoch]
	}
	if mlp.mse[epoch] > mlp.ymax {
		mlp.ymax = mlp.mse[epoch]
	}
}

// determineClass determines testing example class given sample number and sample
func (mlp *MLP) determineClass(j int, sample Sample) error {
	// At output layer, classify example, increment class count, %correct

	// convert node outputs to the class; zero is the threshold
	class := 0
	for i, output := range mlp.node[mlp.hiddenLayers+1] {
		if output.y > 0.0 {
			class |= (1 << i)
		}
	}

	// store classification
	mlp.samples[j].class = class

	// Assign Stats.correct, Stats.classCount
	mlp.statistics.classCount[sample.desired]++
	if class == sample.desired {
		mlp.statistics.correct[class]++
	}

	return nil
}

// class2desired constructs the desired output from the given class
func (mlp *MLP) class2desired(class int) {
	// tranform int to slice of -1 and 1 representing the 0 and 1 bits
	for i := 0; i < len(mlp.desired); i++ {
		if class&1 == 1 {
			mlp.desired[i] = 1
		} else {
			mlp.desired[i] = -1
		}
		class >>= 1
	}
}

func (mlp *MLP) propagateForward(samp Sample, epoch int) error {
	// Assign sample to input layer
	for i := 1; i < len(mlp.node[0]); i++ {
		mlp.node[0][i].y = float64(samp.encChar[i-1])
	}

	// calculate desired from the class
	mlp.class2desired(samp.desired)

	// Loop over layers: mlp.hiddenLayers + output layer
	// input->first hidden, then hidden->hidden,..., then hidden->output
	for layer := 1; layer <= mlp.hiddenLayers; layer++ {
		// Loop over nodes in the layer, d1 is the layer depth of current
		d1 := len(mlp.node[layer])
		for i1 := 1; i1 < d1; i1++ { // this layer loop
			// Each node in previous layer is connected to current node because
			// the network is fully connected.  d2 is the layer depth of previous
			d2 := len(mlp.node[layer-1])
			// Loop over weights to get v
			v := 0.0
			for i2 := 0; i2 < d2; i2++ { // previous layer loop
				v += mlp.link[layer-1][i2*(d1-1)+i1-1].wgt * mlp.node[layer-1][i2].y
			}
			// compute output y = Phi(v)
			mlp.node[layer][i1].y = a * math.Tanh(b*v)
		}
	}

	// last layer is different because there is no bias node, so the indexing is different
	layer := mlp.hiddenLayers + 1
	d1 := len(mlp.node[layer])
	for i1 := 0; i1 < d1; i1++ { // this layer loop
		// Each node in previous layer is connected to current node because
		// the network is fully connected.  d2 is the layer depth of previous
		d2 := len(mlp.node[layer-1])
		// Loop over weights to get v
		v := 0.0
		for i2 := 0; i2 < d2; i2++ { // previous layer loop
			v += mlp.link[layer-1][i2*d1+i1].wgt * mlp.node[layer-1][i2].y
		}
		// compute output y = Phi(v)
		mlp.node[layer][i1].y = a * math.Tanh(b*v)
	}

	return nil
}

func (mlp *MLP) propagateBackward() error {

	// output layer is different, no bias node, so the indexing is different
	// Loop over nodes in output layer
	layer := mlp.hiddenLayers + 1
	d1 := len(mlp.node[layer])
	for i1 := 0; i1 < d1; i1++ { // this layer loop
		//compute error e=d-Phi(v)
		mlp.node[layer][i1].delta = mlp.desired[i1] - mlp.node[mlp.hiddenLayers+1][i1].y
		// Multiply error by this node's Phi'(v) to get local gradient.
		mlp.node[layer][i1].delta *= K1 * (K2 - mlp.node[layer][i1].y*mlp.node[layer][i1].y)
		// Send this node's local gradient to previous layer nodes through corresponding link.
		// Each node in previous layer is connected to current node because the network
		// is fully connected.  d2 is the previous layer depth
		d2 := len(mlp.node[layer-1])
		for i2 := 0; i2 < d2; i2++ { // previous layer loop
			mlp.node[layer-1][i2].delta += mlp.link[layer-1][i2*d1+i1].wgt * mlp.node[layer][i1].delta
			// Compute weight delta, Update weight with momentum, y, and local gradient
			wgtDelta := mlp.learningRate * mlp.node[layer][i1].delta * mlp.node[layer-1][i2].y
			mlp.link[layer-1][i2*d1+i1].wgt +=
				wgtDelta + mlp.momentum*mlp.link[layer-1][i2*d1+i1].wgtDelta
			// update weight delta
			mlp.link[layer-1][i2*d1+i1].wgtDelta = wgtDelta

		}
		// Reset this local gradient to zero for next training example
		mlp.node[layer][i1].delta = 0.0
	}

	// Loop over layers in backward direction, starting at the last hidden layer
	for layer := mlp.hiddenLayers; layer > 0; layer-- {
		// Loop over nodes in this layer, d1 is the current layer depth
		d1 := len(mlp.node[layer])
		for i1 := 1; i1 < d1; i1++ { // this layer loop
			// Multiply deltas propagated from past node by this node's Phi'(v) to get local gradient.
			mlp.node[layer][i1].delta *= K1 * (K2 - mlp.node[layer][i1].y*mlp.node[layer][i1].y)
			// Send this node's local gradient to previous layer nodes through corresponding link.
			// Each node in previous layer is connected to current node because the network
			// is fully connected.  d2 is the previous layer depth
			d2 := len(mlp.node[layer-1])
			for i2 := 0; i2 < d2; i2++ { // previous layer loop
				mlp.node[layer-1][i2].delta += mlp.link[layer-1][i2*(d1-1)+i1-1].wgt * mlp.node[layer][i1].delta
				// Compute weight delta, Update weight with momentum, y, and local gradient
				// anneal learning rate parameter: mlp.learnRate/(epoch*layer)
				// anneal momentum: momentum/(epoch*layer)
				wgtDelta := mlp.learningRate * mlp.node[layer][i1].delta * mlp.node[layer-1][i2].y
				mlp.link[layer-1][i2*(d1-1)+i1-1].wgt +=
					wgtDelta + mlp.momentum*mlp.link[layer-1][i2*(d1-1)+i1-1].wgtDelta
				// update weight delta
				mlp.link[layer-1][i2*(d1-1)+i1-1].wgtDelta = wgtDelta

			}
			// Reset this local gradient to zero for next training example
			mlp.node[layer][i1].delta = 0.0
		}
	}
	return nil
}

// runEpochs performs forward and backward propagation over each sample
func (mlp *MLP) runEpochs() error {

	// Initialize the weights

	// input layer
	// initialize the wgt and wgtDelta randomly, zero mean, normalize by fan-in
	for i := range mlp.link[0] {
		mlp.link[0][i].wgt = 2.0 * (rand.ExpFloat64() - .5) / float64(imageSize+1)
		mlp.link[0][i].wgtDelta = 2.0 * (rand.ExpFloat64() - .5) / float64(imageSize+1)
	}

	// output layer links
	for i := range mlp.link[mlp.hiddenLayers] {
		mlp.link[mlp.hiddenLayers][i].wgt = 2.0 * (rand.Float64() - .5) / float64(mlp.layerDepth)
		mlp.link[mlp.hiddenLayers][i].wgtDelta = 2.0 * (rand.Float64() - .5) / float64(mlp.layerDepth)
	}

	// hidden layers
	for lay := 1; lay < len(mlp.link)-1; lay++ {
		for link := 0; link < len(mlp.link[lay]); link++ {
			mlp.link[lay][link].wgt = 2.0 * (rand.Float64() - .5) / float64(mlp.layerDepth)
			mlp.link[lay][link].wgtDelta = 2.0 * (rand.Float64() - .5) / float64(mlp.layerDepth)
		}
	}
	for n := 0; n < mlp.epochs; n++ {
		//fmt.Printf("epoch %d\n", n)
		// Loop over the training examples
		for _, samp := range mlp.samples {
			// Forward Propagation
			err := mlp.propagateForward(samp, n)
			if err != nil {
				return fmt.Errorf("forward propagation error: %s", err.Error())
			}

			// Backward Propagation
			err = mlp.propagateBackward()
			if err != nil {
				return fmt.Errorf("backward propagation error: %s", err.Error())
			}
		}

		// At the end of each epoch, loop over the output nodes and calculate mse
		mlp.calculateMSE(n)

		// Shuffle training exmaples
		rand.Shuffle(len(mlp.samples), func(i, j int) {
			mlp.samples[i], mlp.samples[j] = mlp.samples[j], mlp.samples[i]
		})

	}

	return nil
}

// init parses the html template files
func init() {
	tmplTrainingMLP = template.Must(template.ParseFiles(fileTrainingMLP))
	tmplTestingMLP = template.Must(template.ParseFiles(fileTestingMLP))
}

// createExamples creates a slice of training or testing examples
func (mlp *MLP) createExamples() error {
	// read in encoded characters from encoded characters .csv file
	f, err := os.Open(path.Join(dataDir, filechars))
	if err != nil {
		fmt.Printf("Error opening file %s: %v\n", filechars, err.Error())
		return fmt.Errorf("open file %s error: %v", filechars, err.Error())
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	class := 0
	for scanner.Scan() {
		line := scanner.Text()
		items := strings.Split(line, ",")
		mlp.char2Class[items[0]] = class
		mlp.samples[class] = Sample{
			encChar: make([]int, len(items)-1),
			desired: class,
		}
		for i, strval := range items[1:] {
			intval, err := strconv.Atoi(strval)
			if err != nil {
				fmt.Printf("String %s to int conversion error: %v\n", strval, err.Error())
				return fmt.Errorf("string %s to int conversion error: %v", strval, err.Error())
			}
			mlp.samples[class].encChar[i] = intval
		}
		class++
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("bufio.Scanner error: %v\n", err.Error())
		return fmt.Errorf("bufio.Scanner error: %v", err.Error())
	}
	return nil
}

// newMLP constructs an MLP instance for training
func newMLP(r *http.Request, hiddenLayers int, plot *PlotT) (*MLP, error) {
	// Read the training parameters in the HTML Form

	txt := r.FormValue("layerdepth")
	layerDepth, err := strconv.Atoi(txt)
	if err != nil {
		fmt.Printf("layerdepth int conversion error: %v\n", err)
		return nil, fmt.Errorf("layerdepth int conversion error: %s", err.Error())
	}

	txt = r.FormValue("learningrate")
	learningRate, err := strconv.ParseFloat(txt, 64)
	if err != nil {
		fmt.Printf("learningrate float conversion error: %v\n", err)
		return nil, fmt.Errorf("learningrate float conversion error: %s", err.Error())
	}

	txt = r.FormValue("momentum")
	momentum, err := strconv.ParseFloat(txt, 64)
	if err != nil {
		fmt.Printf("momentum float conversion error: %v\n", err)
		return nil, fmt.Errorf("momentum float conversion error: %s", err.Error())
	}

	txt = r.FormValue("epochs")
	epochs, err := strconv.Atoi(txt)
	if err != nil {
		fmt.Printf("epochs int conversion error: %v\n", err)
		return nil, fmt.Errorf("epochs int conversion error: %s", err.Error())
	}

	mlp := MLP{
		rows:         300,
		columns:      300,
		hiddenLayers: hiddenLayers,
		layerDepth:   layerDepth,
		epochs:       epochs,
		learningRate: learningRate,
		momentum:     momentum,
		plot:         plot,
		Endpoints: Endpoints{
			ymin: math.MaxFloat64,
			ymax: -math.MaxFloat64,
			xmin: 0,
			xmax: float64(epochs - 1)},
		samples:    make([]Sample, classes),
		char2Class: make(map[string]int),
	}

	// construct link that holds the weights and weight deltas
	mlp.link = make([][]Link, hiddenLayers+1)

	// input layer
	mlp.link[0] = make([]Link, (imageSize+1)*layerDepth)

	// outer layer nodes
	olnodes := int(math.Ceil(math.Log2(float64(classes))))

	// output layer links
	mlp.link[len(mlp.link)-1] = make([]Link, olnodes*(layerDepth+1))

	// hidden layer links
	for i := 1; i < len(mlp.link)-1; i++ {
		mlp.link[i] = make([]Link, (layerDepth+1)*layerDepth)
	}

	// construct node, init node[i][0].y to 1.0 (bias)
	mlp.node = make([][]Node, hiddenLayers+2)

	// input layer
	mlp.node[0] = make([]Node, imageSize+1)
	// set first node in the layer (bias) to 1
	mlp.node[0][0].y = 1.0

	// output layer, which has no bias node
	mlp.node[hiddenLayers+1] = make([]Node, olnodes)

	// hidden layers
	for i := 1; i <= hiddenLayers; i++ {
		mlp.node[i] = make([]Node, layerDepth+1)
		// set first node in the layer (bias) to 1
		mlp.node[i][0].y = 1.0
	}

	// construct desired from classes, binary representation
	mlp.desired = make([]float64, olnodes)

	// mean-square error
	mlp.mse = make([]float64, epochs)

	return &mlp, nil
}

// gridFillInterp inserts the data points in the grid and draws a straight line between points
func (mlp *MLP) gridFillInterp() error {
	var (
		x            float64 = 0.0
		y            float64 = mlp.mse[0]
		prevX, prevY float64
		xscale       float64
		yscale       float64
	)

	// Mark the data x-y coordinate online at the corresponding
	// grid row/column.

	// Calculate scale factors for x and y
	xscale = float64(mlp.columns-1) / (mlp.xmax - mlp.xmin)
	yscale = float64(mlp.rows-1) / (mlp.ymax - mlp.ymin)

	mlp.plot.Grid = make([]string, mlp.rows*mlp.columns)

	// This cell location (row,col) is on the line
	row := int((mlp.ymax-y)*yscale + .5)
	col := int((x-mlp.xmin)*xscale + .5)
	mlp.plot.Grid[row*mlp.columns+col] = "online"

	prevX = x
	prevY = y

	// Scale factor to determine the number of interpolation points
	lenEPy := mlp.ymax - mlp.ymin
	lenEPx := mlp.xmax - mlp.xmin

	// Continue with the rest of the points in the file
	for i := 1; i < mlp.epochs; i++ {
		x++
		// ensemble average of the mse
		y = mlp.mse[i]

		// This cell location (row,col) is on the line
		row := int((mlp.ymax-y)*yscale + .5)
		col := int((x-mlp.xmin)*xscale + .5)
		mlp.plot.Grid[row*mlp.columns+col] = "online"

		// Interpolate the points between previous point and current point

		/* lenEdge := math.Sqrt((x-prevX)*(x-prevX) + (y-prevY)*(y-prevY)) */
		lenEdgeX := math.Abs((x - prevX))
		lenEdgeY := math.Abs(y - prevY)
		ncellsX := int(float64(mlp.columns) * lenEdgeX / lenEPx) // number of points to interpolate in x-dim
		ncellsY := int(float64(mlp.rows) * lenEdgeY / lenEPy)    // number of points to interpolate in y-dim
		// Choose the biggest
		ncells := ncellsX
		if ncellsY > ncells {
			ncells = ncellsY
		}

		stepX := (x - prevX) / float64(ncells)
		stepY := (y - prevY) / float64(ncells)

		// loop to draw the points
		interpX := prevX
		interpY := prevY
		for i := 0; i < ncells; i++ {
			row := int((mlp.ymax-interpY)*yscale + .5)
			col := int((interpX-mlp.xmin)*xscale + .5)
			mlp.plot.Grid[row*mlp.columns+col] = "online"
			interpX += stepX
			interpY += stepY
		}

		// Update the previous point with the current point
		prevX = x
		prevY = y
	}
	return nil
}

// insertLabels inserts x- an y-axis labels in the plot
func (mlp *MLP) insertLabels() {
	mlp.plot.Xlabel = make([]string, xlabels)
	mlp.plot.Ylabel = make([]string, ylabels)
	// Construct x-axis labels
	incr := (mlp.xmax - mlp.xmin) / (xlabels - 1)
	x := mlp.xmin
	// First label is empty for alignment purposes
	for i := range mlp.plot.Xlabel {
		mlp.plot.Xlabel[i] = fmt.Sprintf("%.2f", x)
		x += incr
	}

	// Construct the y-axis labels
	incr = (mlp.ymax - mlp.ymin) / (ylabels - 1)
	y := mlp.ymin
	for i := range mlp.plot.Ylabel {
		mlp.plot.Ylabel[i] = fmt.Sprintf("%.2f", y)
		y += incr
	}
}

// handleTraining performs forward and backward propagation to calculate the weights
func handleTrainingMLP(w http.ResponseWriter, r *http.Request) {

	var (
		plot PlotT
		mlp  *MLP
	)

	// Get the number of hidden layers
	txt := r.FormValue("hiddenlayers")
	// Need hidden layers to continue
	if len(txt) > 0 {
		hiddenLayers, err := strconv.Atoi(txt)
		if err != nil {
			fmt.Printf("Hidden Layers int conversion error: %v\n", err)
			plot.Status = fmt.Sprintf("Hidden Layers conversion to int error: %v", err.Error())
			// Write to HTTP using template and grid
			if err := tmplTrainingMLP.Execute(w, plot); err != nil {
				log.Fatalf("Write to HTTP output using template with error: %v\n", err)
			}
			return
		}

		// create MLP instance to hold state
		mlp, err = newMLP(r, hiddenLayers, &plot)
		if err != nil {
			fmt.Printf("newMLP() error: %v\n", err)
			plot.Status = fmt.Sprintf("newMLP() error: %v", err.Error())
			// Write to HTTP using template and grid
			if err := tmplTrainingMLP.Execute(w, plot); err != nil {
				log.Fatalf("Write to HTTP output using template with error: %v\n", err)
			}
			return
		}

		// Create training examples by reading in the encoded characters
		err = mlp.createExamples()
		if err != nil {
			fmt.Printf("createExamples error: %v\n", err)
			plot.Status = fmt.Sprintf("createExamples error: %v", err.Error())
			// Write to HTTP using template and grid
			if err := tmplTrainingMLP.Execute(w, plot); err != nil {
				log.Fatalf("Write to HTTP output using template with error: %v\n", err)
			}
			return
		}

		// Loop over the Epochs
		err = mlp.runEpochs()
		if err != nil {
			fmt.Printf("runEnsembles() error: %v\n", err)
			plot.Status = fmt.Sprintf("runEnsembles() error: %v", err.Error())
			// Write to HTTP using template and grid
			if err := tmplTrainingMLP.Execute(w, plot); err != nil {
				log.Fatalf("Write to HTTP output using template with error: %v\n", err)
			}
			return
		}

		// Put MSE vs Epoch in PlotT
		err = mlp.gridFillInterp()
		if err != nil {
			fmt.Printf("gridFillInterp() error: %v\n", err)
			plot.Status = fmt.Sprintf("gridFillInterp() error: %v", err.Error())
			// Write to HTTP using template and grid
			if err := tmplTrainingMLP.Execute(w, plot); err != nil {
				log.Fatalf("Write to HTTP output using template with error: %v\n", err)
			}
			return
		}

		// insert x-labels and y-labels in PlotT
		mlp.insertLabels()

		// At the end of all epochs, insert form previous control items in PlotT
		mlp.plot.HiddenLayers = strconv.Itoa(mlp.hiddenLayers)
		mlp.plot.LayerDepth = strconv.Itoa(mlp.layerDepth)
		mlp.plot.Classes = strconv.Itoa(classes)
		mlp.plot.LearningRate = strconv.FormatFloat(mlp.learningRate, 'f', 3, 64)
		mlp.plot.Momentum = strconv.FormatFloat(mlp.momentum, 'f', 3, 64)
		mlp.plot.Epochs = strconv.Itoa(mlp.epochs)

		// Save hidden layers, hidden layer depth, and weights to csv file, one layer per line
		f, err := os.Create(path.Join(dataDir, fileweights))
		if err != nil {
			fmt.Printf("os.Create() file %s error: %v\n", path.Join(fileweights), err)
			plot.Status = fmt.Sprintf("os.Create() file %s error: %v", path.Join(fileweights), err.Error())
			// Write to HTTP using template and grid
			if err := tmplTrainingMLP.Execute(w, plot); err != nil {
				log.Fatalf("Write to HTTP output using template with error: %v\n", err)
			}
			return
		}
		defer f.Close()
		fmt.Fprintf(f, "%d,%d\n",
			mlp.hiddenLayers, mlp.layerDepth)
		for _, layer := range mlp.link {
			for _, node := range layer {
				fmt.Fprintf(f, "%f,", node.wgt)
			}
			fmt.Fprintln(f)
		}

		mlp.plot.Status = "MSE plotted"

		// Execute data on HTML template
		if err = tmplTrainingMLP.Execute(w, mlp.plot); err != nil {
			log.Fatalf("Write to HTTP output using template with error: %v\n", err)
		}
	} else {
		plot.Status = "Enter Multilayer Perceptron (MLP) training parameters."
		// Write to HTTP using template and grid
		if err := tmplTrainingMLP.Execute(w, plot); err != nil {
			log.Fatalf("Write to HTTP output using template with error: %v\n", err)
		}

	}
}

// Classify test examples and display test results
func (mlp *MLP) runClassification() error {
	// Loop over the training examples
	mlp.plot.Grid = make([]string, mlp.rows*mlp.columns)
	mlp.statistics =
		Stats{correct: make([]int, classes), classCount: make([]int, classes)}
	for i, samp := range mlp.samples {
		// Forward Propagation
		err := mlp.propagateForward(samp, 1)
		if err != nil {
			return fmt.Errorf("forward propagation error: %s", err.Error())
		}
		// At output layer, classify example, increment class count, %correct
		// Convert node output y to class
		err = mlp.determineClass(i, samp)
		if err != nil {
			return fmt.Errorf("determineClass error: %s", err.Error())
		}
	}

	mlp.plot.TestResults = make([]Results, classes)

	totalCount := 0
	totalCorrect := 0
	// tabulate TestResults by converting numbers to string in Results
	for i := range mlp.plot.TestResults {
		totalCount += mlp.statistics.classCount[i]
		totalCorrect += mlp.statistics.correct[i]
		mlp.plot.TestResults[i] = Results{
			Class:   strconv.Itoa(i),
			Count:   strconv.Itoa(mlp.statistics.classCount[i]),
			Correct: strconv.Itoa(mlp.statistics.correct[i] * 100 / mlp.statistics.classCount[i]),
		}
	}
	mlp.plot.TotalCount = strconv.Itoa(totalCount)
	mlp.plot.TotalCorrect = strconv.Itoa(totalCorrect * 100 / totalCount)

	mlp.plot.HiddenLayers = strconv.Itoa(mlp.hiddenLayers)
	mlp.plot.LayerDepth = strconv.Itoa(mlp.layerDepth)
	mlp.plot.Classes = strconv.Itoa(classes)

	mlp.plot.Status = "Results completed."

	return nil
}

// drawCharacters draws the characters that are used
func (mlp *MLP) drawCharacters() error {
	// open encoded character file
	// read in encoded characters from encoded characters .csv file
	f, err := os.Open(path.Join(dataDir, filechars))
	if err != nil {
		fmt.Printf("Error opening file %s: %v\n", filechars, err.Error())
		return fmt.Errorf("open file %s error: %v", filechars, err.Error())
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)

	charRows := int(math.Sqrt(classes))
	charCols := int(math.Sqrt(classes))
	const (
		nextRow   = 17
		nextCol   = 17
		startRow  = 8
		startCol  = 8
		imageSize = 81
	)
	endRow := startRow + nextRow*charRows
	endCol := startCol + nextCol*charCols
	imageRow := int(math.Sqrt(imageSize))
	imageCol := int(math.Sqrt(imageSize))

	class := 0

done:
	// loop over rows
	for row := startRow; row < endRow; row += nextRow {
		// loop over columns
		for col := startCol; col < endCol; col += nextCol {
			if ok := scanner.Scan(); !ok {
				break done
			}
			line := scanner.Text()
			item := strings.Split(line, ",")
			// insert this character in TestResults
			mlp.plot.TestResults[class].Character = item[0]
			current := row*mlp.columns + col
			// draw the character
			k := 1
			for i := 0; i < imageRow; i++ {
				for j := 0; j < imageCol; j++ {
					// This cell is part of the character
					if item[k] == "1" {
						mlp.plot.Grid[current+j] = "online"
					}
					k++
				}
				current += mlp.columns
			}

			class++
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("bufio.Scanner error: %v\n", err.Error())
		return fmt.Errorf("bufio.Scanner error: %v", err.Error())
	}
	return nil
}

// newTestingMLP constructs an MLP from the saved mlp weights and parameters
func newTestingMLP(plot *PlotT) (*MLP, error) {
	// Read in weights from csv file, ordered by layers, and MLP parameters
	f, err := os.Open(path.Join(dataDir, fileweights))
	if err != nil {
		fmt.Printf("Open file %s error: %v", fileweights, err)
		return nil, fmt.Errorf("open file %s error: %s", fileweights, err.Error())
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// get the parameters
	scanner.Scan()
	line := scanner.Text()

	items := strings.Split(line, ",")
	hiddenLayers, err := strconv.Atoi(items[0])
	if err != nil {
		fmt.Printf("Conversion to int of %s error: %v", items[0], err)
		return nil, err
	}
	hidLayersDepth, err := strconv.Atoi(items[1])
	if err != nil {
		fmt.Printf("Conversion to int of %s error: %v", items[1], err)
		return nil, err
	}

	// construct the mlp
	mlp := MLP{
		rows:         150,
		columns:      150,
		hiddenLayers: hiddenLayers,
		layerDepth:   hidLayersDepth,
		plot:         plot,
		samples:      make([]Sample, classes),
		char2Class:   make(map[string]int),
	}

	// retrieve the weights
	rows := 0
	for scanner.Scan() {
		rows++
		line = scanner.Text()
		weights := strings.Split(line, ",")
		weights = weights[:len(weights)-1]
		temp := make([]Link, len(weights))
		for i, wtStr := range weights {
			wt, err := strconv.ParseFloat(wtStr, 64)
			if err != nil {
				fmt.Printf("ParseFloat of %s error: %v", wtStr, err)
				continue
			}
			temp[i] = Link{wgt: wt, wgtDelta: 0}
		}
		mlp.link = append(mlp.link, temp)
	}
	if err = scanner.Err(); err != nil {
		fmt.Printf("scanner error: %s", err.Error())
	}

	fmt.Printf("hidden layer depth = %d, hidden layers = %d, classes = %d\n",
		mlp.layerDepth, mlp.hiddenLayers, classes)

	// construct node, init node[i][0].y to 1.0 (bias)
	mlp.node = make([][]Node, mlp.hiddenLayers+2)

	// input layer
	mlp.node[0] = make([]Node, imageSize+1)
	// set first node in the layer (bias) to 1
	mlp.node[0][0].y = 1.0

	// outer layer nodes
	olnodes := int(math.Ceil(math.Log2(float64(classes))))

	// output layer, which has no bias node
	mlp.node[mlp.hiddenLayers+1] = make([]Node, olnodes)

	// hidden layers
	for i := 1; i <= mlp.hiddenLayers; i++ {
		mlp.node[i] = make([]Node, mlp.layerDepth+1)
		// set first node in the layer (bias) to 1
		mlp.node[i][0].y = 1.0
	}

	// construct desired from classes, binary representation
	mlp.desired = make([]float64, olnodes)

	return &mlp, nil
}

// handleTesting performs pattern classification of the test data
func handleTestingMLP(w http.ResponseWriter, r *http.Request) {
	var (
		plot PlotT
		mlp  *MLP
		err  error
	)
	// Construct MLP instance containing MLP state
	mlp, err = newTestingMLP(&plot)
	if err != nil {
		fmt.Printf("newTestingMLP() error: %v\n", err)
		plot.Status = fmt.Sprintf("newTestingMLP() error: %v", err.Error())
		// Write to HTTP using template and grid
		if err := tmplTrainingMLP.Execute(w, plot); err != nil {
			log.Fatalf("Write to HTTP output using template with error: %v\n", err)
		}
		return
	}

	// Create testing examples by reading in the encoded characters
	err = mlp.createExamples()
	if err != nil {
		fmt.Printf("createExamples error: %v\n", err)
		plot.Status = fmt.Sprintf("createExamples error: %v", err.Error())
		// Write to HTTP using template and grid
		if err := tmplTrainingMLP.Execute(w, plot); err != nil {
			log.Fatalf("Write to HTTP output using template with error: %v\n", err)
		}
		return
	}

	// At end of all examples tabulate TestingResults
	// Convert numbers to string in Results
	err = mlp.runClassification()
	if err != nil {
		fmt.Printf("runClassification() error: %v\n", err)
		plot.Status = fmt.Sprintf("runClassification() error: %v", err.Error())
		// Write to HTTP using template and grid
		if err := tmplTrainingMLP.Execute(w, plot); err != nil {
			log.Fatalf("Write to HTTP output using template with error: %v\n", err)
		}
		return
	}

	// Draw the characters to show what is being classified
	err = mlp.drawCharacters()
	if err != nil {
		fmt.Printf("drawCharacters() error: %v\n", err)
		plot.Status = fmt.Sprintf("drawCharacters() error: %v", err.Error())
		// Write to HTTP using template and grid
		if err := tmplTrainingMLP.Execute(w, plot); err != nil {
			log.Fatalf("Write to HTTP output using template with error: %v\n", err)
		}
		return
	}

	// Execute data on HTML template
	if err = tmplTestingMLP.Execute(w, mlp.plot); err != nil {
		log.Fatalf("Write to HTTP output using template with error: %v\n", err)
	}
}

// executive creates the HTTP handlers, listens and serves
func main() {
	// Set up HTTP servers with handlers for training and testing the MLP Neural Network

	// Create HTTP handler for training
	http.HandleFunc(patternTrainingMLP, handleTrainingMLP)
	// Create HTTP handler for testing
	http.HandleFunc(patternTestingMLP, handleTestingMLP)
	fmt.Printf("Multilayer Perceptron Neural Network Server listening on %v.\n", addr)
	http.ListenAndServe(addr, nil)
}
