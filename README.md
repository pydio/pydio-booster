# Pydio Booster

Golang tools to improve admin experience on both deployment and performance.

Download binaries directly at :

    https://download.pydio.com/pub/booster

Dependencies : 
    
    git (https://git-scm.com/book/en/v2/Getting-Started-Installing-Git)
    go (https://golang.org/doc/install)
    glide (https://glide.sh/)

To install from source : 

    git clone https://github.com/pydio/go.git ${GOPATH}/src/github.com/pydio/go
    ${GOPATH}/src/github.com/pydio/pydio-booster/build/install.sh

To build for a machine : 

    git clone https://github.com/pydio/go.git ${GOPATH}/src/github.com/pydio/go
    ${GOPATH}/src/github.com/pydio/pydio-booster/build/build.sh --machines "darwin|386" --dir ${GOPATH}/bin --force