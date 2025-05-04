@echo off
set SRC_DIR=src
set BUILD_DIR=build
set BINARY_NAME=family_site
set HTML_DIR=html
set STATIC_DIR=static
set NAS_USER=steven
set NAS_IP=192.168.1.46
set NAS_DEST=/var/services/homes/Steven/Site
set NAS_SERVICE=family_site

set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0

if not exist %BUILD_DIR% (
	mkdir "%BUILD_DIR%"
)

cd "%SRC_DIR%"

 
go build -ldflags="-X main.isProd=true" -o "../%BUILD_DIR%/%BINARY_NAME%"

cd ".."

if %errorlevel% neq 0 (
	echo Build failed!
	pause
	exit /b %errorlevel%
)

echo Build completed successfully

echo Stopping the service %NAS_SERVICE% on the NAS...
ssh %NAS_USER%@%NAS_IP% "./stop_service.sh"

if %errorlevel% neq 0 (
    echo Failed to stop the service!
    pause
    exit /b %errorlevel%
)

echo Deploying %BINARY_NAME% to NAS as %NAS_IP%...
scp -O "%BUILD_DIR%\%BINARY_NAME%" %NAS_USER%@%NAS_IP%:%NAS_DEST%

if %errorlevel% neq 0 (
	echo Deployment failed
	pause
	exit /b %errorlevel%
)

echo Deploying %HTML_DIR% to NAS at %NAS_IP%...
scp -O -r "%HTML_DIR%" %NAS_USER%@%NAS_IP%:%NAS_DEST%/

if %errorlevel% neq 0 (
    echo Deployment of HTML folder failed
    pause
    exit /b %errorlevel%
)

echo Deploying %STATIC_DIR% to NAS at %NAS_IP%...
scp -O -r "%STATIC_DIR%" %NAS_USER%@%NAS_IP%:%NAS_DEST%/

if %errorlevel% neq 0 (
    echo Deployment of static folder failed
    pause
    exit /b %errorlevel%
)

echo Starting the service %NAS_SERVICE% on the NAS...
ssh %NAS_USER%@%NAS_IP% "./start_service.sh"

echo Deployed successfully.
pause