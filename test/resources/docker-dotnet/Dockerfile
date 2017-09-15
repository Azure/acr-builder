FROM microsoft/aspnetcore:2.0 AS base
WORKDIR /app
EXPOSE 80

FROM microsoft/aspnetcore-build:2.0 AS builder
WORKDIR /src
COPY *.sln ./
COPY Web/Web.csproj Web/
RUN dotnet restore
COPY . .
WORKDIR /src/Web
RUN dotnet build -c Release -o /app

FROM builder AS publish
RUN dotnet publish -c Release -o /app

FROM imaginary/cert-generator:1.0
RUN generate-cert.sh

FROM base AS production
WORKDIR /app
COPY --from=publish /app .
COPY --from=3 /cert /app
ENTRYPOINT ["dotnet", "Web.dll"]
