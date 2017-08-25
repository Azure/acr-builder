#!/bin/sh

dockerComposeVersion=$1
targetLocation=$2

get_docker_compose() {
    until wget -O $targetLocation "https://github.com/docker/compose/releases/download/${dockerComposeVersion}/docker-compose-`uname -s`-`uname -m`"
    do
        sleep 10
    done
    # make it executabe
    chmod +x $targetLocation
}

verify_get_docker_compose()
{
    # we'll verify that the file we're getting is an executable linkable format file, 64-bit
    file $targetLocation | grep "ELF 64-bit LSB"
    while [ $? -ne 0 ];
    do
        get_docker_compose
        file $targetLocation | grep "ELF 64-bit LSB"
    done
}

verify_get_docker_compose
