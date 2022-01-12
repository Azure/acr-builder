# Required.
# docker build -f baseimages/docker-cli/Windows.Dockerfile -t docker .

ARG WINDOWS_IMAGE=mcr.microsoft.com/windows/servercore:ltsc2022
FROM $WINDOWS_IMAGE as base

# set the default shell as powershell.
# $ProgressPreference: https://github.com/PowerShell/PowerShell/issues/2138#issuecomment-251261324
SHELL ["powershell", "-Command", "$ErrorActionPreference = 'Stop'; $ProgressPreference = 'SilentlyContinue';"]

# install MinGit (especially for "go get" and docker build by git repos)
ENV GIT_VERSION 2.17.1
ENV GIT_TAG v${GIT_VERSION}.windows.1
ENV GIT_DOWNLOAD_URL https://github.com/git-for-windows/git/releases/download/${GIT_TAG}/MinGit-${GIT_VERSION}-64-bit.zip
ENV GIT_DOWNLOAD_SHA256 668d16a799dd721ed126cc91bed49eb2c072ba1b25b50048280a4e2c5ed56e59
# disable prompt asking for credential
ENV GIT_TERMINAL_PROMPT 0
RUN Write-Host ('Downloading {0} ...' -f $env:GIT_DOWNLOAD_URL); \
	[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12; \
	Invoke-WebRequest -Uri $env:GIT_DOWNLOAD_URL -OutFile 'git.zip'; \
	\
	Write-Host 'Expanding ...'; \
	Expand-Archive -Path git.zip -DestinationPath C:\git\.; \
	\
	Write-Host 'Removing ...'; \
	Remove-Item git.zip -Force; \
	\
	Write-Host 'Updating PATH ...'; \
	$env:PATH = 'C:\git\cmd;C:\git\mingw64\bin;C:\git\usr\bin;' + $env:PATH; \
	[Environment]::SetEnvironmentVariable('PATH', $env:PATH, [EnvironmentVariableTarget]::Machine); \
	\
	Write-Host 'Verifying install ...'; \
	Write-Host 'git --version'; git --version; \
	\
	Write-Host 'Complete.';

# install git-lfs
ARG GIT_LFS_VERSION=2.5.2
ENV GIT_LFS_DOWNLOAD_URL https://github.com/git-lfs/git-lfs/releases/download/v${GIT_LFS_VERSION}/git-lfs-windows-amd64-v${GIT_LFS_VERSION}.zip
RUN Write-Host ('Downloading {0} ...' -f $env:GIT_LFS_DOWNLOAD_URL); \
	[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12; \
	Invoke-WebRequest -Uri $env:GIT_LFS_DOWNLOAD_URL -OutFile 'git-lfs.zip'; \
	\
	Write-Host 'Expanding ...'; \
	Expand-Archive -Path git-lfs.zip -DestinationPath C:\git-lfs\.; \
	\
	Write-Host 'Removing ...'; \
	Remove-Item git-lfs.zip -Force; \
	\
	Write-Host 'Updating PATH ...'; \
	$env:PATH = 'C:\git-lfs;' + $env:PATH; \
	[Environment]::SetEnvironmentVariable('PATH', $env:PATH, [EnvironmentVariableTarget]::Machine); \
	\
	Write-Host 'Installing ...'; \
	Write-Host 'git lfs install'; git lfs install --system; \
	\
	Write-Host 'Complete.';

# Download the docker executable
FROM base as dockercli
ARG DOCKER_VERSION=19-03-18
ENV DOCKER_DOWNLOAD_URL https://dockermsft.blob.core.windows.net/dockercontainer/docker-${DOCKER_VERSION}.zip
RUN Write-Host ('Downloading {0} ...' -f $env:DOCKER_DOWNLOAD_URL); \
	[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12; \
	Invoke-WebRequest -Uri $env:DOCKER_DOWNLOAD_URL -OutFile 'docker.zip'; \
	\
	Write-Host 'Expanding ...'; \
	Expand-Archive -Path docker.zip -DestinationPath C:\unzip\.; \
	\
	Write-Host 'Removing dockerd.exe ...'; \
	Remove-Item C:\unzip\docker\dockerd.exe -Force; \
	\
	Write-Host 'Complete.';

# setup the runtime environment
FROM base as runtime
COPY --from=dockercli C:/unzip/docker/ C:/docker/
RUN setx /M PATH $('C:\docker;{0}' -f $env:PATH);
ENTRYPOINT [ "docker.exe" ]
CMD [ "--help" ]