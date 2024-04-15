<h3>Character Recognition using a Multilayer Perceptron Neural Network with the Back-Propagation Algorithm</h3>
<hr>
This program is a web application written in Go that makes extensive use of the html/template package.
Navigate to the C:\Users\your-name\CharacterRecognition\src\backprop\ directory and issue "go run ann.go" to
start the Multilayer Perceptron (MLP) Neural Network server. In a web browser enter http://127.0.0.1:8080/mlpbackprop
in the address bar.  There are two phases of operation:  the training phase and the testing phase.  During the training
phase, examples consisting of x-y coordinates in the Cartesian Plane and the desired class are supplied to the network.
The network itself is a directed graph consisting of an input layer of nodes, one or more hidden layers of nodes, and
an output layer of nodes.  Each layer of nodes can be arbitrarily deep.  The nodes of the network are connected by weighted
links.  The network is fully connected.  This means that every node is connected to its immediately adjacent neighbor node.  The weights are trained
by first propagating the inputs forward, layer by layer, to the output layer of nodes.  The output layer of nodes finds the
difference between the desired and its output and back propagates the errors to the input layer.  The hidden and input layer
weights are assigned “credit” for the errors by using the chain rule of differential calculus.  Each neuron consists of a
linear combiner and an activation function.  This program uses the hyperbolic tangent function to serve as the activation function.
This function is non-linear and differentiable and limits its output to be between -1 and 1.  <b>The purpose of this program is to classify an alpha-numeric
character</b>.
The user selects the MLP training parameters:
<li>Hidden Layers</li>
<li>Layer Depth</li>
<li>Learning Rate</li>
<li>Momentum</li>
<li>Epochs</li>
<br>
<p>
The <i>Learning Rate</i> and <i>Momentum</i> must be less than one.  Each <i>Epoch</i> consists of the number of <i>Training Examples</i>.  
One training example is an alpha-numeric character and the desired class (0, 1,…, 63).  There are 64 characters and therefore 64 classes.
The characters are a sequence of eighty-one 1 and -1 integers that represent the encoded image of the character.  The file containing the
encoded characters was produced by the character encoding repository program at thomasteplick/char-encoder.
</p>
<p>
When the <i>Submit</i> button on the MLP Training Parameters form is clicked, the weights in the network are trained
and the mean-square error (MSE) is graphed versus Epoch.  As can be seen in the screen shots below, there is significant variance over the ensemble,
but it eventually settles down after about 150 epochs. An epoch is the forward and backward propagation of all the 64 training samples.
</p>
<p>
When the <i>Test</i> link is clicked, 64 examples are supplied to the MLP.  It classifies the alpha-numerica characters.
The test results are tabulated and the actual characters are graphed from the encoding that was supplied to the MLP.
It takes some trial-and-error with the MLP Training Parameters to reduce the MSE to zero.  It is possible to a specify a 
more complex MLP than necessary and not get good results.  For example, using more hidden layers, a greater layer depth,
or over training with more examples than necessary may be detrimental to the MLP.  Clicking the <i>Train</i> link starts a new training
phase and the MLP Training Parameters must be entered again.
</p>

<b>Character Recognition Learning Curve, MSE vs Epoch, 3 Hidden Layers, Hidden Layer Depth = 10</b>

![image](https://github.com/thomasteplick/char-recognition/assets/117768679/286c1930-015f-4b0a-95fa-13b8f9721d94)

<b>Character Recognition Test Results, 3 Hidden Layers, Hidden Layer Depth = 10</b>

![image](https://github.com/thomasteplick/char-recognition/assets/117768679/06a3c95d-660f-4cd7-aaae-4a15ed31d36a)
![image](https://github.com/thomasteplick/char-recognition/assets/117768679/a9fb88ef-719c-4993-883c-215914000ab1)
