# Required.
# docker build -f baseimages/docker.Windows.Dockerfile -t docker .
ARG WINDOWS_IMAGE=microsoft/windowsservercore:1803

FROM ${WINDOWS_IMAGE} as download
SHELL ["powershell", "-Command", "$ErrorActionPreference = 'Stop'; $ProgressPreference = 'SilentlyContinue';"]
ENV DOCKER_VERSION 18.03.0-ce

# Utility to extract the Docker toolbox zip
RUN Invoke-WebRequest 'http://constexpr.org/innoextract/files/innoextract-1.6-windows.zip' -OutFile 'innoextract.zip' -UseBasicParsing ; \
    Expand-Archive innoextract.zip -DestinationPath C:\ ; \
    Remove-Item -Path innoextract.zip

RUN [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12 ; \
    Invoke-WebRequest $('https://github.com/docker/toolbox/releases/download/v{0}/DockerToolbox-{0}.exe' -f $env:DOCKER_VERSION) -OutFile 'dockertoolbox.exe' -UseBasicParsing
RUN /innoextract.exe dockertoolbox.exe

FROM ${WINDOWS_IMAGE} as environment
SHELL ["powershell", "-Command", "$ErrorActionPreference = 'Stop'; $ProgressPreference = 'SilentlyContinue';"]
COPY --from=download /app/docker.exe /

# install MinGit (especially for "go get" and docker build by git repos) 
ENV GIT_VERSION 2.17.1
ENV GIT_TAG v${GIT_VERSION}.windows.1
ENV GIT_DOWNLOAD_URL https://github.com/git-for-windows/git/releases/download/${GIT_TAG}/MinGit-${GIT_VERSION}-64-bit.zip
ENV GIT_DOWNLOAD_SHA256 668d16a799dd721ed126cc91bed49eb2c072ba1b25b50048280a4e2c5ed56e59
RUN Write-Host ('Downloading {0} ...' -f $env:GIT_DOWNLOAD_URL); \
	[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12; \
	Invoke-WebRequest -Uri $env:GIT_DOWNLOAD_URL -OutFile 'git.zip'; \	
	Write-Host 'Expanding ...'; \
	Expand-Archive -Path git.zip -DestinationPath C:\git\.; \
	Write-Host 'Removing ...'; \
	Remove-Item git.zip -Force; \
	Write-Host 'Updating PATH ...'; \
	$env:PATH = 'C:\git\cmd;C:\git\mingw64\bin;C:\git\usr\bin;' + $env:PATH; \
	[Environment]::SetEnvironmentVariable('PATH', $env:PATH, [EnvironmentVariableTarget]::Machine); \
	Write-Host 'Verifying install ...'; \
	Write-Host 'git --version'; git --version; \
	Write-Host 'Complete.';

CMD [ "docker.exe" ]