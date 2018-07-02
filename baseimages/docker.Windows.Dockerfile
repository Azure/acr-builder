ARG NANOSERVER_IMAGE="sac2016"

# Required.
# docker build -f baseimages/docker.Windows.Dockerfile -t docker .
FROM microsoft/windowsservercore as download

SHELL ["powershell", "-Command", "$ErrorActionPreference = 'Stop'; $ProgressPreference = 'SilentlyContinue';"]

ENV DOCKER_VERSION 18.03.0-ce

# Utility to extract the Docker toolbox zip
RUN Invoke-WebRequest 'http://constexpr.org/innoextract/files/innoextract-1.6-windows.zip' -OutFile 'innoextract.zip' -UseBasicParsing ; \
    Expand-Archive innoextract.zip -DestinationPath C:\ ; \
    Remove-Item -Path innoextract.zip

RUN [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12 ; \
    Invoke-WebRequest $('https://github.com/docker/toolbox/releases/download/v{0}/DockerToolbox-{0}.exe' -f $env:DOCKER_VERSION) -OutFile 'dockertoolbox.exe' -UseBasicParsing
RUN /innoextract.exe dockertoolbox.exe

FROM microsoft/nanoserver:$NANOSERVER_IMAGE
COPY --from=download /app/docker.exe /
CMD [ "docker.exe" ]