## Lab 1 - Interaction with Remote Objects, Go version

This file details the contents of the initial Lab 1 code repository and how to use it.


### Getting started

If you're using the same programming language for this lab as the previous one, the look and feel of this
lab should be familiar, and your environment shouldn't need any changes.  If this is your first lab using Go,
all you should need to install is a recent Go compiler.  For example, a default Ubuntu Linux installation
would only require you to do
```
sudo snap install go --classic
go env -w GO111MODULE=auto
```
to start working on the lab. Everything else is controlled by `make` using the `go` command and shell scripts. For
more details about installing `go`, see [https://go.dev/doc/install](https://go.dev/doc/install).


### Initial repository contents

The top-level directory (called `lab1-go` here) of the initial starter-code repository includes three things:
* This `README.md` file
* The Lab 1 `Makefile`, described in detail later
* The `src` source directory, which is where all your work will be done

Visually, this repository looks roughly like the following:
```
\---lab1-go
        +---src
        |   \---remote
        |   |   +---remote.go
        |   |   \---remote_test.go
        +---Makefile
        \---README.md
```
The details of each of these will hopefully become clear after reading the rest of this file.


### Building your remote library

The `remote` package initially only includes the basic structure of the library components, though a few things are included in their entirety for your use.  There is a lot of initial documentation in the `.go` files to help you out, and we strongly recommend starting there.  Your remote package can live entirely in a single `remote.go` file, or you can split content into multiple files as desired, as long as the entire `remote` directory can be copied into other projects in the future and function as a self-contained library.  The `remote_test.go` file includes the automated tests that will be used to verify the requirements of your library and determine your total score for the coding component of this lab.


### Testing the remote library

When you're ready to test the components of your library, use can either use the appropriate `go test` commands or the provided `make` rules `checkpoint`, `final`, and `all` as were used in the previous lab.  The `Makefile` also includes rules that run Go's race detector to ensure that your library is thread safe.  These rules are `checkpoint-race`, `final-race`, and `all-race`; these rules are used by the auto-grader.

Because our expected outcome in this lab is a library, it's a little more tricky to test than our game server was in Lab 0. However, since we're again using sockets for interaction between components, you are free to use similar approaches and test tools that you used previously (e.g., `nc`). Printing/logging status and error messages can be very helpful, but make sure to remove any such functionality from your code before making your final submission (unless you can't pass all the tests and want to include it in support of partial credit).

You are welcome to create additional `make` rules in the Makefile, but we ask that you keep the existing `final` and `checkpoint`
rules, as we will use them for lab grading.


### Example application

In the later part of the lab, once your `remote` package is complete, you'll build your own custom application that imports the external library.  Your application should be created in a new folder within the `src` directory (at the same level as the `remote` directory), and it should include at least two separate files (since they would really be executed on different machines over a network) for the client and server application code, each of which can be executed using `go run` or `go build` followed by manual execution of the resulting binary.  You are welcome to add new rules to the `Makefile` for launching these different application components, but this is not necessary as long as the commands to run them are clearly documented in your submission.  However, good documentation is expected for the application.  In addition to standard documentation, you will need to write a small usage guide for how the application works.


### Generating documentation

We want you to get in the habit of documenting your code in a way that leads to detailed, easy-to-read/-navigate package documentation for your code package. Our Makefile includes a `docs` rule that will pipe the output of the `go doc` command into a text file that is reasonably easy to read.  We will use this output for the manually graded parts of the lab, so good comments are valuable.


### Questions?

If there is any part of the initial repository, environment setup, lab requirements, or anything else, please do not hesitate
to ask.  We're here to help!

