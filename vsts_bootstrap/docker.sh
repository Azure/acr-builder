docker version
docker login -u $1 -p $2 $3
docker pull $4
docker pull $5
docker run -v /var/run/docker.sock:/var/run/docker.sock $4 --docker-repository $6 --azure-storage-account $7 --azure-account-key $8 --checkout-root $9
