#!/bin/bash
# POSIX

# Static
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PKG="github.com/pydio/pydio-booster"

# IO 
dir="${GOPATH}/bin"
file="${GOPATH}/src/${PKG}/build/machines.txt"

# Variables
machines=()
done=()
force=false

# Show the help message
show_help() {
    echo "build.sh [-m|--machines machine...] [-d|--dir DIR] [-f|--force]" >&2
    exit 0
}

# Read
readmachines() {
    while IFS='' read -r line; do
        read os arch <<<$line
	machines+=("$os|$arch")
    done < $1

}

# Cancel process and cleanup
cancel() {
    popd 2>& 1>/dev/null
    rm -rf "${dir}/${CURRENT_OS}_${CURRENT_ARCH}"

    echo ""

    exit $1
}
trap 'cancel 0' INT

error() {
    local line="$1"
    local message="$2"
    local code="${3:-1}"

    echo -ne "Error line ${line}: ${message}"
    cancel "${code}"
}
trap 'error ${LINENO}' ERR

while :; do
    case $1 in
	-m|--machines)
            if [ -n "$2" ]; then
		if [ -f "$2" ]; then
                    readmachines $2
                else
                    machines+=($2)
                fi

	        shift
		shift
	    else
	        printf 'ERROR: "--machines" requires a non-empty option argument.\n' >&2
	        exit 1
	    fi
            ;;
        -d|--dir)
            if [ -n "$2" ]; then
	        dir=$2
	        shift
	    else
	        printf 'ERROR: "--file" requires a non-empty option argument.\n' >&2
	        exit 1
	    fi
            ;;
        -f|--force)
            force=true
	    shift
            ;;
	-h|-\?|--help)
	    show_help
            exit
            ;;
	-v|--verbose)
	    verbose=$((verbose + 1)) # Each -v argument adds 1 to verbosity.
	    ;;
	--) # End of all options.
	    shift
	    break
 	    ;;
	-?*)
	    printf 'WARN: Unknown option (ignored): %s\n' "$1" >&2
	    ;;
	*) # Default case: If no more options then break out of the loop.
	    break
    esac
done

if [ ! -d "${GOPATH}/src/${PKG}/cmd/pydio" ]; then
    echo "${GOPATH}/src/${PKG}/cmd/pydio not found"
    echo "Run : go get github.com/pydio/pydio-booster"
    exit 1
fi

echo "========================="
echo "PYDIO GO Build process"
echo " - Directory $DIR"
echo "========================="
echo ""

echo -ne "  - Getting build dependencies\n";
echo -ne "    ";
${DIR}/install.sh
echo -ne "OK\n\n";

for machine in "${machines[@]}"; do
    IFS="|" read os arch <<<"$machine"

    CURRENT_OS=$os
    CURRENT_ARCH=$arch

    echo -ne "  - Starting build for $os $arch\n";
    echo -ne "    ";

    if [ -d "${dir}/${os}_${arch}" ]; then
        if [ $force = true ]; then
	    rm -rf ${dir}/${os}_${arch}
            echo -ne "Old Build removed\n";
            echo -ne "    ";
        else
            echo -ne "Ignoring - Dir already exists\n\n"
            continue
	fi;
    fi

    mkdir -p "${dir}/${os}_${arch}" 1>/dev/null
    pushd "${dir}/${os}_${arch}" 1>/dev/null

    env GOOS=$os GOARCH=$arch go build -v ${PKG}/cmd/pydio 1>/dev/null

    popd 1>/dev/null

    echo -ne "OK\n\n";
done
