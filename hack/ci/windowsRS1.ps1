# Jenkins CI scripts for Windows to Windows CI (Powershell Version)
# By John Howard (@jhowardmsft) January 2016 - bash version; July 2016 Ported to PowerShell

$ErrorActionPreference = 'Stop'
$StartTime=Get-Date
#$env:DOCKER_DUT_DEBUG="yes" # Comment out to not be in debug mode

# -------------------------------------------------------------------------------------------
# When executed, we rely on four variables being set in the environment:
#
# [The reason for being environment variables rather than parameters is historical. No reason
# why it couldn't be updated.]
#
#    SOURCES_DRIVE       is the drive on which the sources being tested are cloned from.
#                        This should be a straight drive letter, no platform semantics.
#                        For example 'c'
#
#    SOURCES_SUBDIR      is the top level directory under SOURCES_DRIVE where the
#                        sources are cloned to. There are no platform semantics in this
#                        as it does not include slashes.
#                        For example 'gopath'
#
#                        Based on the above examples, it would be expected that Jenkins
#                        would clone the sources being tested to
#                        SOURCES_DRIVE\SOURCES_SUBDIR\src\github.com\docker\docker, or
#                        c:\gopath\src\github.com\docker\docker
#
#    TESTRUN_DRIVE       is the drive where we build the binary on and redirect everything
#                        to for the daemon under test. On an Azure D2 type host which has
#                        an SSD temporary storage D: drive, this is ideal for performance.
#                        For example 'd'
#
#    TESTRUN_SUBDIR      is the top level directory under TESTRUN_DRIVE where we redirect
#                        everything to for the daemon under test. For example 'CI'.
#                        Hence, the daemon under test is run under
#                        TESTRUN_DRIVE\TESTRUN_SUBDIR\CI-<CommitID> or
#                        d:\CI\CI-<CommitID>
#
# In addition, the following variables can control the run configuration:
#
#    DOCKER_DUT_DEBUG         if defined starts the daemon under test in debug mode.
#
#    SKIP_VALIDATION_TESTS    if defined skips the validation tests
#
#    SKIP_UNIT_TESTS          if defined skips the unit tests
#
#    SKIP_INTEGRATION_TESTS   if defined skips the integration tests
#
#    SKIP_COPY_GO             if defined skips copy the go installer from the image
#
#    DOCKER_DUT_HYPERV        if default daemon under test default isolation is hyperv
#
#    INTEGRATION_TEST_NAME    to only run partial tests eg "TestInfo*" will only run
#                             any tests starting "TestInfo"
#
#    SKIP_BINARY_BUILD        if defined skips building the binary
#
#    SKIP_ZAP_DUT             if defined doesn't zap the daemon under test directory
#
#    SKIP_IMAGE_BUILD         if defined doesn't build the 'docker' image
#
#    INTEGRATION_IN_CONTAINER if defined, runs the integration tests from inside a container.
#                             As of July 2016, there are known issues with this.
#
#    SKIP_ALL_CLEANUP         if defined, skips any cleanup at the start or end of the run
#
#    WINDOWS_BASE_IMAGE       if defined, uses that as the base image. Note that the
#                             docker integration tests are also coded to use the same
#                             environment variable, and if no set, defaults to microsoft/windowsservercore
# -------------------------------------------------------------------------------------------
#
# Jenkins Integration. Add a Windows Powershell build step as follows:
#
#    Write-Host -ForegroundColor green "INFO: Jenkins build step starting"
#    $CISCRIPT_DEFAULT_LOCATION = "https://raw.githubusercontent.com/jhowardmsft/docker-w2wCIScripts/master/runCI/executeCI.ps1"
#    $CISCRIPT_LOCAL_LOCATION = "$env:TEMP\executeCI.ps1"
#    Write-Host -ForegroundColor green "INFO: Removing cached execution script"
#    Remove-Item $CISCRIPT_LOCAL_LOCATION -Force -ErrorAction SilentlyContinue 2>&1 | Out-Null
#    $wc = New-Object net.webclient
#    try {
#        Write-Host -ForegroundColor green "INFO: Downloading latest execution script..."
#        $wc.Downloadfile($CISCRIPT_DEFAULT_LOCATION, $CISCRIPT_LOCAL_LOCATION)
#    }
#    catch [System.Net.WebException]
#    {
#        Throw ("Failed to download: $_")
#    }
#    & $CISCRIPT_LOCAL_LOCATION
# -------------------------------------------------------------------------------------------

$SCRIPT_VER="10-May-2017 12:16 PDT"
$FinallyColour="Cyan"

#$env:SKIP_UNIT_TESTS="yes"
#$env:SKIP_VALIDATION_TESTS="yes"
#$env:SKIP_ZAP_DUT=""
#$env:SKIP_BINARY_BUILD="yes"
#$env:INTEGRATION_TEST_NAME=""
#$env:SKIP_IMAGE_BUILD="yes"
#$env:SKIP_ALL_CLEANUP="yes"
#$env:INTEGRATION_IN_CONTAINER="yes"
#$env:WINDOWS_BASE_IMAGE=""
#$env:SKIP_COPY_GO="yes"

Function Nuke-Everything {
    $ErrorActionPreference = 'SilentlyContinue'
    if ($env:SKIP_ALL_CLEANUP -ne $null) {
        Write-Host -ForegroundColor Magenta "WARN: Skipping all cleanup"
        return
    }

    try {
        Write-Host -ForegroundColor green "INFO: Nuke-Everything..."
        $containerCount = ($(docker ps -aq | Measure-Object -line).Lines)
        if (-not $LastExitCode -eq 0) {
            Throw "ERROR: Failed to get container count from control daemon while nuking"
        }

        Write-Host -ForegroundColor green "INFO: Container count on control daemon to delete is $containerCount"
        if ($(docker ps -aq | Measure-Object -line).Lines -gt 0) {
            docker rm -f $(docker ps -aq)
        }
        $imageCount=($(docker images --format "{{.Repository}}:{{.ID}}" | `
                        select-string -NotMatch "windowsservercore" | `
                        select-string -NotMatch "nanoserver" | `
                        select-string -NotMatch "docker" | `
                        Measure-Object -line).Lines)
        if ($imageCount -gt 0) {
            Write-Host -Foregroundcolor green "INFO: Non-base image count on control daemon to delete is $imageCount"
            docker rmi -f `
                $(docker images --format "{{.Repository}}:{{.ID}}" | `
                        select-string -NotMatch "windowsservercore" | `
                        select-string -NotMatch "nanoserver" | `
                        select-string -NotMatch "docker").ToString().Split(":")[1]
        }

        # Kill any spurious daemons. The '-' is IMPORTANT otherwise will kill the control daemon!
        $pids=$(get-process | where-object {$_.ProcessName -like 'moby-engine-*'}).id
        foreach ($p in $pids) {
            Write-Host "INFO: Killing daemon with PID $p"
            Stop-Process -Id $p -Force -ErrorAction SilentlyContinue
        }

        Stop-Process -name "cc1" -Force -ErrorAction SilentlyContinue 2>&1 | Out-Null
        Stop-Process -name "link" -Force -ErrorAction SilentlyContinue 2>&1 | Out-Null
        Stop-Process -name "compile" -Force -ErrorAction SilentlyContinue 2>&1 | Out-Null
        Stop-Process -name "ld" -Force -ErrorAction SilentlyContinue 2>&1 | Out-Null
        Stop-Process -name "go" -Force -ErrorAction SilentlyContinue 2>&1 | Out-Null
        Stop-Process -name "git" -Force -ErrorAction SilentlyContinue 2>&1 | Out-Null
        Stop-Process -name "git-remote-https" -Force -ErrorAction SilentlyContinue 2>&1 | Out-Null
        Stop-Process -name "integration-cli.test" -Force -ErrorAction SilentlyContinue 2>&1 | Out-Null
        Stop-Process -name "tail" -Force -ErrorAction SilentlyContinue 2>&1 | Out-Null

        # Detach any VHDs
        gwmi msvm_mountedstorageimage -namespace root/virtualization/v2 -ErrorAction SilentlyContinue | foreach-object {$_.DetachVirtualHardDisk() }

        # Stop any compute processes
        Get-ComputeProcess | Stop-ComputeProcess -Force

        # Delete the directory using our dangerous utility unless told not to
        if (Test-Path "$env:TESTRUN_DRIVE`:\$env:TESTRUN_SUBDIR") {
            if ($env:SKIP_ZAP_DUT -eq $null) {
                Write-Host -ForegroundColor Green "INFO: Nuking $env:TESTRUN_DRIVE`:\$env:TESTRUN_SUBDIR"
                docker-ci-zap "-folder=$env:TESTRUN_DRIVE`:\$env:TESTRUN_SUBDIR"
            } else {
                Write-Host -ForegroundColor Magenta "WARN: Skip nuking $env:TESTRUN_DRIVE`:\$env:TESTRUN_SUBDIR"
            }
        }

        # RS1 Production Server workaround - Psched
        $reg = "HKLM:\System\CurrentControlSet\Services\Psched\Parameters\NdisAdapters"
        $count=(Get-ChildItem $reg | Measure-Object).Count
        if ($count -gt 0) {
            Write-Warning "There are $count NdisAdapters leaked under Psched\Parameters"
            if ($env:COMPUTERNAME -match "jenkins-rs1-") {
                Write-Warning "Cleaning Psched..."
                Get-ChildItem $reg | Remove-Item -Recurse -Force -ErrorAction SilentlyContinue | Out-Null
            } else {
                Write-Warning "Not cleaning as not a production RS1 server"
            }
        }

        # RS1 Production Server workaround - WFPLWFS
        $reg = "HKLM:\System\CurrentControlSet\Services\WFPLWFS\Parameters\NdisAdapters"
        $count=(Get-ChildItem $reg | Measure-Object).Count
        if ($count -gt 0) {
            Write-Warning "There are $count NdisAdapters leaked under WFPLWFS\Parameters"
            if ($env:COMPUTERNAME -match "jenkins-rs1-") {
                Write-Warning "Cleaning WFPLWFS..."
                Get-ChildItem $reg | Remove-Item -Recurse -Force -ErrorAction SilentlyContinue | Out-Null
            } else {
                Write-Warning "Not cleaning as not a production RS1 server"
            }
        }
    } catch {
        # Don't throw any errors onwards Throw $_
    }
}

Try {
    Write-Host -ForegroundColor Cyan "`nINFO: executeCI.ps1 starting at $(date)`n"
    Write-Host  -ForegroundColor Green "INFO: Script version $SCRIPT_VER"
    Set-PSDebug -Trace 0  # 1 to turn on
    $origPath="$env:PATH"            # so we can restore it at the end
    $origDOCKER_HOST="$DOCKER_HOST"  # So we can restore it at the end
    $origGOROOT="$env:GOROOT"        # So we can restore it at the end
    $origGOPATH="$env:GOPATH"        # So we can restore it at the end

    # Turn off progress bars
    $origProgressPreference=$global:ProgressPreference
    $global:ProgressPreference='SilentlyContinue'

    # Git version
    Write-Host  -ForegroundColor Green "INFO: Running $(git version)"

    # OS Version
    $bl=(Get-ItemProperty -Path "Registry::HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion"  -Name BuildLabEx).BuildLabEx
    $a=$bl.ToString().Split(".")
    $Branch=$a[3]
    $WindowsBuild=$a[0]+"."+$a[1]+"."+$a[4]
    Write-Host -ForegroundColor green "INFO: Branch:$Branch Build:$WindowsBuild"

    # List the environment variables
    Write-Host -ForegroundColor green "INFO: Environment variables:"
    Get-ChildItem Env: | Out-String

    # PR
    if (-not ($env:PR -eq $Null)) { echo "INFO: PR#$env:PR (https://github.com/docker/docker/pull/$env:PR)" }

    # Make sure docker is installed
    if ((Get-Command "docker" -ErrorAction SilentlyContinue) -eq $null) { Throw "ERROR: docker is not installed or not found on path" }

    # Make sure docker-ci-zap is installed
    if ((Get-Command "docker-ci-zap" -ErrorAction SilentlyContinue) -eq $null) { Throw "ERROR: docker-ci-zap is not installed or not found on path" }

    # Make sure SOURCES_DRIVE is set
    if ($env:SOURCES_DRIVE -eq $Null) { Throw "ERROR: Environment variable SOURCES_DRIVE is not set" }

    # Make sure TESTRUN_DRIVE is set
    if ($env:TESTRUN_DRIVE -eq $Null) { Throw "ERROR: Environment variable TESTRUN_DRIVE is not set" }

    # Make sure SOURCES_SUBDIR is set
    if ($env:SOURCES_SUBDIR -eq $Null) { Throw "ERROR: Environment variable SOURCES_SUBDIR is not set" }

    # Make sure TESTRUN_SUBDIR is set
    if ($env:TESTRUN_SUBDIR -eq $Null) { Throw "ERROR: Environment variable TESTRUN_SUBDIR is not set" }

    # SOURCES_DRIVE\SOURCES_SUBDIR must be a directory and exist
    if (-not (Test-Path -PathType Container "$env:SOURCES_DRIVE`:\$env:SOURCES_SUBDIR")) { Throw "ERROR: $env:SOURCES_DRIVE`:\$env:SOURCES_SUBDIR must be an existing directory" }


    # Create the TESTRUN_DRIVE\TESTRUN_SUBDIR if it does not already exist
    New-Item -ItemType Directory -Force -Path "$env:TESTRUN_DRIVE`:\$env:TESTRUN_SUBDIR" -ErrorAction SilentlyContinue | Out-Null

    Write-Host  -ForegroundColor Green "INFO: Sources under $env:SOURCES_DRIVE`:\$env:SOURCES_SUBDIR\..."
    Write-Host  -ForegroundColor Green "INFO: Test run under $env:TESTRUN_DRIVE`:\$env:TESTRUN_SUBDIR\..."

    # Check the intended source location is a directory
    if (-not (Test-Path -PathType Container "$env:SOURCES_DRIVE`:\$env:SOURCES_SUBDIR\src\github.com\docker\docker" -ErrorAction SilentlyContinue)) {
        Throw "ERROR: $env:SOURCES_DRIVE`:\$env:SOURCES_SUBDIR\src\github.com\docker\docker is not a directory!"
    }

    # Make sure we start at the root of the sources
    cd "$env:SOURCES_DRIVE`:\$env:SOURCES_SUBDIR\src\github.com\docker\docker"
    Write-Host  -ForegroundColor Green "INFO: Running in $(pwd)"

    # Make sure we are in repo
    if (-not (Test-Path -PathType Leaf -Path ".\Dockerfile.windows")) {
        Throw "$(pwd) does not container Dockerfile.Windows!"
    }
    Write-Host  -ForegroundColor Green "INFO: docker/docker repository was found"

    # Make sure microsoft/windowsservercore:latest image is installed in the control daemon. On public CI machines, windowsservercore.tar and nanoserver.tar
    # are pre-baked and tagged appropriately in the c:\baseimages directory, and can be directly loaded.
    # Note - this script will only work on 10B (Oct 2016) or later machines! Not 9D or previous due to image tagging assumptions.
    #
    # On machines not on Microsoft corpnet, or those which have not been pre-baked, we have to docker pull the image in which case it will
    # will come in directly as microsoft/windowsservercore:latest. The ultimate goal of all this code is to ensure that whatever,
    # we have microsoft/windowsservercore:latest
    #
    # Note we cannot use (as at Oct 2016) nanoserver as the control daemons base image, even if nanoserver is used in the tests themselves.

    $ErrorActionPreference = "SilentlyContinue"
    $ControlDaemonBaseImage="windowsservercore"

    $readBaseFrom="c"
    if ($((docker images --format "{{.Repository}}:{{.Tag}}" | Select-String $("microsoft/"+$ControlDaemonBaseImage+":latest") | Measure-Object -Line).Lines) -eq 0) {
        # Try the internal azure CI image version or Microsoft internal corpnet where the base image is already pre-prepared on the disk,
        # either through Invoke-DockerCI or, in the case of Azure CI servers, baked into the VHD at the same location.
        if (Test-Path $("$env:SOURCES_DRIVE`:\baseimages\"+$ControlDaemonBaseImage+".tar")) {

            # An optimization for CI servers to copy it to the D: drive which is an SSD.
            if ($env:SOURCES_DRIVE -ne $env:TESTRUN_DRIVE) {
                $readBaseFrom=$env:TESTRUN_DRIVE
                if (!(Test-Path "$env:TESTRUN_DRIVE`:\baseimages")) {
                    New-Item "$env:TESTRUN_DRIVE`:\baseimages" -type directory | Out-Null
                }
                if (!(Test-Path "$env:TESTRUN_DRIVE`:\baseimages\windowsservercore.tar")) {
                    if (Test-Path "$env:SOURCES_DRIVE`:\baseimages\windowsservercore.tar") {
                        Write-Host -ForegroundColor Green "INFO: Optimisation - copying $env:SOURCES_DRIVE`:\baseimages\windowsservercore.tar to $env:TESTRUN_DRIVE`:\baseimages"
                        Copy-Item "$env:SOURCES_DRIVE`:\baseimages\windowsservercore.tar" "$env:TESTRUN_DRIVE`:\baseimages"
                    }
                }
                if (!(Test-Path "$env:TESTRUN_DRIVE`:\baseimages\nanoserver.tar")) {
                    if (Test-Path "$env:SOURCES_DRIVE`:\baseimages\nanoserver.tar") {
                        Write-Host -ForegroundColor Green "INFO: Optimisation - copying $env:SOURCES_DRIVE`:\baseimages\nanoserver.tar to $env:TESTRUN_DRIVE`:\baseimages"
                        Copy-Item "$env:SOURCES_DRIVE`:\baseimages\nanoserver.tar" "$env:TESTRUN_DRIVE`:\baseimages"
                    }
                }
                $readBaseFrom=$env:TESTRUN_DRIVE
            }

            Write-Host  -ForegroundColor Green "INFO: Loading"$ControlDaemonBaseImage".tar from disk. This may take some time..."
            $ErrorActionPreference = "SilentlyContinue"
            docker load -i $("$readBaseFrom`:\baseimages\"+$ControlDaemonBaseImage+".tar")
            $ErrorActionPreference = "Stop"
            if (-not $LastExitCode -eq 0) {
                Throw $("ERROR: Failed to load $readBaseFrom`:\baseimages\"+$ControlDaemonBaseImage+".tar")
            }
            Write-Host -ForegroundColor Green "INFO: docker load of"$ControlDaemonBaseImage" completed successfully"
        } else {
            # We need to docker pull it instead. It will come in directly as microsoft/imagename:latest
            Write-Host -ForegroundColor Green $("INFO: Pulling microsoft/"+$ControlDaemonBaseImage+":latest from docker hub. This may take some time...")
            $ErrorActionPreference = "SilentlyContinue"
            docker pull $("microsoft/"+$ControlDaemonBaseImage)
            $ErrorActionPreference = "Stop"
            if (-not $LastExitCode -eq 0) {
                Throw $("ERROR: Failed to docker pull microsoft/"+$ControlDaemonBaseImage+":latest.")
            }
            Write-Host -ForegroundColor Green $("INFO: docker pull of microsoft/"+$ControlDaemonBaseImage+":latest completed successfully")
        }
    } else {
        Write-Host -ForegroundColor Green "INFO: Image"$("microsoft/"+$ControlDaemonBaseImage+":latest")"is already loaded in the control daemon"
    }

    # Inspect the pulled image to get the version directly
    $ErrorActionPreference = "SilentlyContinue"
    $imgVersion = $(docker inspect  $("microsoft/"+$ControlDaemonBaseImage) --format "{{.OsVersion}}")
    $ErrorActionPreference = "Stop"
    Write-Host -ForegroundColor Green $("INFO: Version of microsoft/"+$ControlDaemonBaseImage+":latest is '"+$imgVersion+"'")

    # Provide the docker version for debugging purposes.
    Write-Host  -ForegroundColor Green "INFO: Docker version of control daemon"
    Write-Host
    $ErrorActionPreference = "SilentlyContinue"
    docker version
    $ErrorActionPreference = "Stop"
    if (-not($LastExitCode -eq 0)) {
        Write-Host
        Write-Host  -ForegroundColor Green "---------------------------------------------------------------------------"
        Write-Host  -ForegroundColor Green " Failed to get a response from the control daemon. It may be down."
        Write-Host  -ForegroundColor Green " Try re-running this CI job, or ask on #docker-dev or #docker-maintainers"
        Write-Host  -ForegroundColor Green " to see if the the daemon is running. Also check the nssm configuration."
        Write-Host  -ForegroundColor Green " DOCKER_HOST is set to $DOCKER_HOST."
        Write-Host  -ForegroundColor Green "---------------------------------------------------------------------------"
        Write-Host
        Throw "ERROR: The control daemon does not appear to be running."
    }
    Write-Host

    # Same as above, but docker info
    Write-Host  -ForegroundColor Green "INFO: Docker info of control daemon"
    Write-Host
    $ErrorActionPreference = "SilentlyContinue"
    docker info
    $ErrorActionPreference = "Stop"
    if (-not($LastExitCode -eq 0)) {
        Throw "ERROR: The control daemon does not appear to be running."
    }
    Write-Host

    # Get the commit has and verify we have something
    $ErrorActionPreference = "SilentlyContinue"
    $COMMITHASH=$(git rev-parse --short HEAD)
    $ErrorActionPreference = "Stop"
    if (-not($LastExitCode -eq 0)) {
        Throw "ERROR: Failed to get commit hash. Are you sure this is a docker repository?"
    }
    Write-Host  -ForegroundColor Green "INFO: Commit hash is $COMMITHASH"

    # Nuke everything and go back to our sources after
    Nuke-Everything
    cd "$env:SOURCES_DRIVE`:\$env:SOURCES_SUBDIR\src\github.com\docker\docker"

    # Redirect to a temporary location.
    $TEMPORIG=$env:TEMP
    $env:TEMP="$env:TESTRUN_DRIVE`:\$env:TESTRUN_SUBDIR\CI-$COMMITHASH"
    $env:LOCALAPPDATA="$TEMP\localappdata"
    $errorActionPreference='Stop'
    New-Item -ItemType Directory "$env:TEMP" -ErrorAction SilentlyContinue | Out-Null
    New-Item -ItemType Directory "$env:TEMP\userprofile" -ErrorAction SilentlyContinue  | Out-Null
    New-Item -ItemType Directory "$env:TEMP\localappdata" -ErrorAction SilentlyContinue | Out-Null
    New-Item -ItemType Directory "$env:TEMP\binary" -ErrorAction SilentlyContinue | Out-Null
    New-Item -ItemType Directory "$env:TEMP\installer" -ErrorAction SilentlyContinue | Out-Null
    if ($env:SKIP_COPY_GO -eq $null) {
        # Wipe the previous version of GO - we're going to get it out of the image
        if (Test-Path "$env:TEMP\go") { Remove-Item "$env:TEMP\go" -Recurse -Force -ErrorAction SilentlyContinue | Out-Null }
        New-Item -ItemType Directory "$env:TEMP\go" -ErrorAction SilentlyContinue | Out-Null
    }

    Write-Host -ForegroundColor Green "INFO: Location for testing is $env:TEMP"

    # CI Integrity check - ensure Dockerfile.windows and Dockerfile go versions match
    $goVersionDockerfileWindows=$(Get-Content ".\Dockerfile.windows" | Select-String "ENV GO_VERSION").ToString().Replace("ENV GO_VERSION=","").Replace("\","").Replace("``","").Trim()
    $goVersionDockerfile=$(Get-Content ".\Dockerfile" | Select-String "ENV GO_VERSION").ToString().Split(" ")[2]
    Write-Host  -ForegroundColor Green "INFO: Validating GOLang consistency in Dockerfile.windows..."
    if (-not ($goVersionDockerfile -eq $goVersionDockerfileWindows)) {
        Throw "ERROR: Mismatched GO versions between Dockerfile and Dockerfile.windows. Update your PR to ensure that both files are updated and in sync. $goVersionDockerfile $goVersionDockerfileWindows"
    }

    # Build the image
    if ($env:SKIP_IMAGE_BUILD -eq $null) {
        Write-Host  -ForegroundColor Cyan "`n`nINFO: Building the image from Dockerfile.windows at $(Get-Date)..."
        Write-Host
        $ErrorActionPreference = "SilentlyContinue"
        $Duration=$(Measure-Command { docker build -t docker -f Dockerfile.windows . | Out-Host })
        $ErrorActionPreference = "Stop"
        if (-not($LastExitCode -eq 0)) {
           Throw "ERROR: Failed to build image from Dockerfile.windows"
        }
        Write-Host  -ForegroundColor Green "INFO: Image build ended at $(Get-Date). Duration`:$Duration"
    } else {
        Write-Host -ForegroundColor Magenta "WARN: Skipping building the docker image"
    }

    # DON'T THINK THIS IS USED ANYMORE.... $v=$(Get-Content ".\VERSION" -raw).ToString().Replace("`n","").Trim()
    $contPath="$COMMITHASH`:c`:\go\src\github.com\docker\docker\bundles"

    # After https://github.com/docker/docker/pull/30290, .git was added to .dockerignore. Therefore
    # we have to calculate unsupported outside of the container, and pass the commit ID in through
    # an environment variable for the binary build
    $CommitUnsupported=""
    if ($(git status --porcelain --untracked-files=no).Length -ne 0) {
        $CommitUnsupported="-unsupported"
    }

    # Build the binary in a container unless asked to skip it.
    if ($env:SKIP_BINARY_BUILD -eq $null) {
        Write-Host  -ForegroundColor Cyan "`n`nINFO: Building the test binaries at $(Get-Date)..."
        $ErrorActionPreference = "SilentlyContinue"
        docker rm -f $COMMITHASH 2>&1 | Out-Null
        if ($CommitUnsupported -ne "") {
            Write-Host ""
            Write-Warning "This version is unsupported because there are uncommitted file(s)."
            Write-Warning "Either commit these changes, or add them to .gitignore."
            git status --porcelain --untracked-files=no | Write-Warning
             Write-Host ""
        }
        $Duration=$(Measure-Command {docker run --name $COMMITHASH -e DOCKER_GITCOMMIT=$COMMITHASH$CommitUnsupported docker hack\make.ps1 -Binary | Out-Host })
        $ErrorActionPreference = "Stop"
        if (-not($LastExitCode -eq 0)) {
            Throw "ERROR: Failed to build binary"
        }
        Write-Host  -ForegroundColor Green "INFO: Binaries build ended at $(Get-Date). Duration`:$Duration"

        # Copy the binaries and the generated version_autogen.go out of the container
        $ErrorActionPreference = "SilentlyContinue"
        docker cp "$contPath\docker.exe" $env:TEMP\binary\
        if (-not($LastExitCode -eq 0)) {
            Throw "ERROR: Failed to docker cp the client binary (docker.exe) to $env:TEMP\binary"
        }
        docker cp "$contPath\moby-engine.exe" $env:TEMP\binary\
        if (-not($LastExitCode -eq 0)) {
            Throw "ERROR: Failed to docker cp the daemon binary (moby-engine.exe) to $env:TEMP\binary"
        }
        $ErrorActionPreference = "Stop"

        # Copy the built moby-engine.exe to moby-engine-$COMMITHASH.exe so that easily spotted in task manager.
        Write-Host -ForegroundColor Green "INFO: Copying the built daemon binary to $env:TEMP\binary\moby-engine-$COMMITHASH.exe..."
        Copy-Item $env:TEMP\binary\moby-engine.exe $env:TEMP\binary\moby-engine-$COMMITHASH.exe -Force -ErrorAction SilentlyContinue

        # Copy the built docker.exe to docker-$COMMITHASH.exe
        Write-Host -ForegroundColor Green "INFO: Copying the built client binary to $env:TEMP\binary\docker-$COMMITHASH.exe..."
        Copy-Item $env:TEMP\binary\docker.exe $env:TEMP\binary\docker-$COMMITHASH.exe -Force -ErrorAction SilentlyContinue

    } else {
        Write-Host -ForegroundColor Magenta "WARN: Skipping building the binaries"
    }

    Write-Host -ForegroundColor Green "INFO: Copying autoversion from the container..."
    $ErrorActionPreference = "SilentlyContinue"
    docker cp "$contPath\..\autoversion\version_autogen.go" "$env:SOURCES_DRIVE`:\$env:SOURCES_SUBDIR\src\github.com\docker\docker\autoversion"
    if (-not($LastExitCode -eq 0)) {
         Throw "ERROR: Failed to docker cp the generated version_autogen.go to $env:SOURCES_DRIVE`:\$env:SOURCES_SUBDIR\src\github.com\docker\docker\autoversion"
    }
    $ErrorActionPreference = "Stop"

    # Grab the golang installer out of the built image. That way, we know we are consistent once extracted and paths set,
    # so there's no need to re-deploy on account of an upgrade to the version of GO being used in docker.
    if ($env:SKIP_COPY_GO -eq $null) {
        Write-Host -ForegroundColor Green "INFO: Copying the golang package from the container to $env:TEMP\installer\go.zip..."
        docker cp "$COMMITHASH`:c`:\go.zip" $env:TEMP\installer\
        if (-not($LastExitCode -eq 0)) {
            Throw "ERROR: Failed to docker cp the golang installer 'go.zip' from container:c:\go.zip to $env:TEMP\installer"
        }
        $ErrorActionPreference = "Stop"

        # Extract the golang installer
        Write-Host -ForegroundColor Green "INFO: Extracting go.zip to $env:TEMP\go"
        $Duration=$(Measure-Command { Expand-Archive $env:TEMP\installer\go.zip $env:TEMP -Force | Out-Null})
        Write-Host  -ForegroundColor Green "INFO: Extraction ended at $(Get-Date). Duration`:$Duration"
    } else {
        Write-Host -ForegroundColor Magenta "WARN: Skipping copying and extracting golang from the image"
    }

    # Set the GOPATH
    Write-Host -ForegroundColor Green "INFO: Updating the golang and path environment variables"
    $env:GOPATH="$env:SOURCES_DRIVE`:\$env:SOURCES_SUBDIR"
    Write-Host -ForegroundColor Green "INFO: GOPATH=$env:GOPATH"

    # Set the path to have the version of go from the image at the front
    $env:PATH="$env:TEMP\go\bin;$env:PATH"

    # Set the GOROOT to be our copy of go from the image
    $env:GOROOT="$env:TEMP\go"
    Write-Host -ForegroundColor Green "INFO: $(go version)"

    # Work out the the -H parameter for the daemon under test (DASHH_DUT) and client under test (DASHH_CUT)
    #$DASHH_DUT="npipe:////./pipe/$COMMITHASH" # Can't do remote named pipe
    #$ip = (resolve-dnsname $env:COMPUTERNAME -type A -NoHostsFile -LlmnrNetbiosOnly).IPAddress # Useful to tie down
    $DASHH_CUT="tcp://127.0.0.1`:2357"    # Not a typo for 2375!
    $DASHH_DUT="tcp://0.0.0.0:2357"       # Not a typo for 2375!

    # Arguments for the daemon under test
    $dutArgs=@()
    $dutArgs += "-H $DASHH_DUT"
    $dutArgs += "--graph $env:TEMP\daemon"
    $dutArgs += "--pidfile $env:TEMP\docker.pid"

    # Arguments: Are we starting the daemon under test in debug mode?
    if (-not ("$env:DOCKER_DUT_DEBUG" -eq "")) {
        Write-Host -ForegroundColor Green "INFO: Running the daemon under test in debug mode"
        $dutArgs += "-D"
    }

    # Arguments: Are we starting the daemon under test with Hyper-V containers as the default isolation?
    if (-not ("$env:DOCKER_DUT_HYPERV" -eq "")) {
        Write-Host -ForegroundColor Green "INFO: Running the daemon under test with Hyper-V containers as the default"
        $dutArgs += "--exec-opt isolation=hyperv"
    }

    # Start the daemon under test, ensuring everything is redirected to folders under $TEMP.
    # Important - we launch the -$COMMITHASH version so that we can kill it without
    # killing the control daemon.
    Write-Host -ForegroundColor Green "INFO: Starting a daemon under test..."
    Write-Host -ForegroundColor Green "INFO: Args: $dutArgs"
    New-Item -ItemType Directory $env:TEMP\daemon -ErrorAction SilentlyContinue  | Out-Null

    # Cannot fathom why, but always writes to stderr....
    Start-Process "$env:TEMP\binary\moby-engine-$COMMITHASH" `
                  -ArgumentList $dutArgs `
                  -RedirectStandardOutput "$env:TEMP\dut.out" `
                  -RedirectStandardError "$env:TEMP\dut.err"
    Write-Host -ForegroundColor Green "INFO: Process started successfully."
    $daemonStarted=1

    # Start tailing the daemon under test if the command is installed
    if ((Get-Command "tail" -ErrorAction SilentlyContinue) -ne $null) {
        $tail = start-process "tail" -ArgumentList "-f $env:TEMP\dut.out" -ErrorAction SilentlyContinue
    }

    # Verify we can get the daemon under test to respond
    $tries=20
    Write-Host -ForegroundColor Green "INFO: Waiting for the daemon under test to start..."
    while ($true) {
        $ErrorActionPreference = "SilentlyContinue"
        & "$env:TEMP\binary\docker-$COMMITHASH" "-H=$($DASHH_CUT)" version 2>&1 | Out-Null
        $ErrorActionPreference = "Stop"
        if ($LastExitCode -eq 0) {
            break
        }

        $tries--
        if ($tries -le 0) {
            $DumpDaemonLog=1
            Throw "ERROR: Failed to get a response from the daemon under test"
        }
        Write-Host -NoNewline "."
        sleep 1
    }
    Write-Host -ForegroundColor Green "INFO: Daemon under test started and replied!"

    # Provide the docker version of the daemon under test for debugging purposes.
    Write-Host -ForegroundColor Green "INFO: Docker version of the daemon under test"
    Write-Host
    $ErrorActionPreference = "SilentlyContinue"
    & "$env:TEMP\binary\docker-$COMMITHASH" "-H=$($DASHH_CUT)" version
    $ErrorActionPreference = "Stop"
    if ($LastExitCode -ne 0) {
        Throw "ERROR: The daemon under test does not appear to be running."
        $DumpDaemonLog=1
    }
    Write-Host

    # Same as above but docker info
    Write-Host -ForegroundColor Green "INFO: Docker info of the daemon under test"
    Write-Host
    $ErrorActionPreference = "SilentlyContinue"
    & "$env:TEMP\binary\docker-$COMMITHASH" "-H=$($DASHH_CUT)" info
    $ErrorActionPreference = "Stop"
    if ($LastExitCode -ne 0) {
        Throw "ERROR: The daemon under test does not appear to be running."
        $DumpDaemonLog=1
    }
    Write-Host

    # Same as above but docker images
    Write-Host -ForegroundColor Green "INFO: Docker images of the daemon under test"
    Write-Host
    $ErrorActionPreference = "SilentlyContinue"
    & "$env:TEMP\binary\docker-$COMMITHASH" "-H=$($DASHH_CUT)" images
    $ErrorActionPreference = "Stop"
    if ($LastExitCode -ne 0) {
        Throw "ERROR: The daemon under test does not appear to be running."
        $DumpDaemonLog=1
    }
    Write-Host

    # Default to windowsservercore for the base image used for the tests. The "docker" image
    # and the control daemon use microsoft/windowsservercore regardless. This is *JUST* for the tests.
    if ($env:WINDOWS_BASE_IMAGE -eq $Null) {
        $env:WINDOWS_BASE_IMAGE="microsoft/windowsservercore"
    }

    # Lowercase and make sure it has a microsoft/ prefix
    $env:WINDOWS_BASE_IMAGE = $env:WINDOWS_BASE_IMAGE.ToLower()
    if ($($env:WINDOWS_BASE_IMAGE -Split "/")[0] -ne "microsoft") {
        Throw "ERROR: WINDOWS_BASE_IMAGE should start microsoft/"
    }

    Write-Host -ForegroundColor Green "INFO: Base image for tests is $env:WINDOWS_BASE_IMAGE"

    $ErrorActionPreference = "SilentlyContinue"
    if ($((& "$env:TEMP\binary\docker-$COMMITHASH" "-H=$($DASHH_CUT)" images --format "{{.Repository}}:{{.Tag}}" | Select-String $($env:WINDOWS_BASE_IMAGE+":latest") | Measure-Object -Line).Lines) -eq 0) {
        # Try the internal azure CI image version or Microsoft internal corpnet where the base image is already pre-prepared on the disk,
        # either through Invoke-DockerCI or, in the case of Azure CI servers, baked into the VHD at the same location.
        if (Test-Path $("c:\baseimages\"+$($env:WINDOWS_BASE_IMAGE -Split "/")[1]+".tar")) {
            Write-Host  -ForegroundColor Green "INFO: Loading"$($env:WINDOWS_BASE_IMAGE -Split "/")[1]".tar from disk into the daemon under test. This may take some time..."
            $ErrorActionPreference = "SilentlyContinue"
            & "$env:TEMP\binary\docker-$COMMITHASH" "-H=$($DASHH_CUT)" load -i $("$readBaseFrom`:\baseimages\"+$($env:WINDOWS_BASE_IMAGE -Split "/")[1]+".tar")
            $ErrorActionPreference = "Stop"
            if (-not $LastExitCode -eq 0) {
                Throw $("ERROR: Failed to load $readBaseFrom`:\baseimages\"+$($env:WINDOWS_BASE_IMAGE -Split "/")[1]+".tar into daemon under test")
            }
            Write-Host -ForegroundColor Green "INFO: docker load of"$($env:WINDOWS_BASE_IMAGE -Split "/")[1]" into daemon under test completed successfully"
        } else {
            # We need to docker pull it instead. It will come in directly as microsoft/imagename:latest
            Write-Host -ForegroundColor Green $("INFO: Pulling "+$env:WINDOWS_BASE_IMAGE+":latest from docker hub into daemon under test. This may take some time...")
            $ErrorActionPreference = "SilentlyContinue"
            & "$env:TEMP\binary\docker-$COMMITHASH" "-H=$($DASHH_CUT)" pull $($env:WINDOWS_BASE_IMAGE)
            $ErrorActionPreference = "Stop"
            if (-not $LastExitCode -eq 0) {
                Throw $("ERROR: Failed to docker pull "+$env:WINDOWS_BASE_IMAGE+":latest into daemon under test.")
            }
            Write-Host -ForegroundColor Green $("INFO: docker pull of "+$env:WINDOWS_BASE_IMAGE+":latest into daemon under test completed successfully")
        }
    } else {
        Write-Host -ForegroundColor Green "INFO: Image"$($env:WINDOWS_BASE_IMAGE+":latest")"is already loaded in the daemon under test"
    }

    # Inspect the pulled or loaded image to get the version directly
    $ErrorActionPreference = "SilentlyContinue"
    $dutimgVersion = $(&"$env:TEMP\binary\docker-$COMMITHASH" "-H=$($DASHH_CUT)" inspect  $($env:WINDOWS_BASE_IMAGE) --format "{{.OsVersion}}")
    $ErrorActionPreference = "Stop"
    Write-Host -ForegroundColor Green $("INFO: Version of "+$env:WINDOWS_BASE_IMAGE+":latest is '"+$dutimgVersion+"'")

    # Run the validation tests unless SKIP_VALIDATION_TESTS is defined.
    if ($env:SKIP_VALIDATION_TESTS -eq $null) {
        Write-Host -ForegroundColor Cyan "INFO: Running validation tests at $(Get-Date)..."
        $ErrorActionPreference = "SilentlyContinue"
        $Duration=$(Measure-Command { hack\make.ps1 -DCO -GoFormat -PkgImports | Out-Host })
        $ErrorActionPreference = "Stop"
        if (-not($LastExitCode -eq 0)) {
            Throw "ERROR: Validation tests failed"
        }
        Write-Host  -ForegroundColor Green "INFO: Validation tests ended at $(Get-Date). Duration`:$Duration"
    } else {
        Write-Host -ForegroundColor Magenta "WARN: Skipping validation tests"
    }

    # Run the unit tests inside a container unless SKIP_UNIT_TESTS is defined
    if ($env:SKIP_UNIT_TESTS -eq $null) {
        Write-Host -ForegroundColor Cyan "INFO: Running unit tests at $(Get-Date)..."
        $ErrorActionPreference = "SilentlyContinue"
        $Duration=$(Measure-Command {docker run -e DOCKER_GITCOMMIT=$COMMITHASH$CommitUnsupported docker hack\make.ps1 -TestUnit | Out-Host })
        $ErrorActionPreference = "Stop"
        if (-not($LastExitCode -eq 0)) {
            Throw "ERROR: Unit tests failed"
        }
        Write-Host  -ForegroundColor Green "INFO: Unit tests ended at $(Get-Date). Duration`:$Duration"
    } else {
        Write-Host -ForegroundColor Magenta "WARN: Skipping unit tests"
    }

    # Add the busybox image. Needed for integration tests
    if ($env:SKIP_INTEGRATION_TESTS -eq $null) {
        $ErrorActionPreference = "SilentlyContinue"
        # Build it regardless while switching between nanoserver and windowsservercore
        #$bbCount = $(& "$env:TEMP\binary\docker-$COMMITHASH" "-H=$($DASHH_CUT)" images | Select-String "busybox" | Measure-Object -line).Lines
        #$ErrorActionPreference = "Stop"
        #if (-not($LastExitCode -eq 0)) {
        #    Throw "ERROR: Could not determine if busybox image is present"
        #}
        #if ($bbCount -eq 0) {
            Write-Host -ForegroundColor Green "INFO: Building busybox"
            $ErrorActionPreference = "SilentlyContinue"

            # This is a temporary hack for nanoserver
            if ($env:WINDOWS_BASE_IMAGE -ne "microsoft/windowsservercore") {
                Write-Host -ForegroundColor Red "HACK HACK HACK - Building 64-bit nanoserver busybox image"
                $(& "$env:TEMP\binary\docker-$COMMITHASH" "-H=$($DASHH_CUT)" build -t busybox https://raw.githubusercontent.com/jhowardmsft/busybox64/master/Dockerfile | Out-Host)
            } else {
                $(& "$env:TEMP\binary\docker-$COMMITHASH" "-H=$($DASHH_CUT)" build -t busybox https://raw.githubusercontent.com/jhowardmsft/busybox/master/Dockerfile | Out-Host)
            }
            $ErrorActionPreference = "Stop"
            if (-not($LastExitCode -eq 0)) {
                Throw "ERROR: Failed to build busybox image"
            }
        #}


        Write-Host -ForegroundColor Green "INFO: Docker images of the daemon under test"
        Write-Host
        $ErrorActionPreference = "SilentlyContinue"
        & "$env:TEMP\binary\docker-$COMMITHASH" "-H=$($DASHH_CUT)" images
        $ErrorActionPreference = "Stop"
        if ($LastExitCode -ne 0) {
            Throw "ERROR: The daemon under test does not appear to be running."
            $DumpDaemonLog=1
        }
        Write-Host
    }

    # Run the integration tests unless SKIP_INTEGRATION_TESTS is defined
    if ($env:SKIP_INTEGRATION_TESTS -eq $null) {
        Write-Host -ForegroundColor Cyan "INFO: Running integration tests at $(Get-Date)..."
        $ErrorActionPreference = "SilentlyContinue"

        # Location of the daemon under test.
        $env:OrigDOCKER_HOST="$env:DOCKER_HOST"

        #https://blogs.technet.microsoft.com/heyscriptingguy/2011/09/20/solve-problems-with-external-command-lines-in-powershell/ is useful to see tokenising
        $c = "go test "
        $c += "`"-check.v`" "
        if ($env:INTEGRATION_TEST_NAME -ne $null) { # Makes is quicker for debugging to be able to run only a subset of the integration tests
            $c += "`"-check.f`" "
            $c += "`"$env:INTEGRATION_TEST_NAME`" "
            Write-Host -ForegroundColor Magenta "WARN: Only running integration tests matching $env:INTEGRATION_TEST_NAME"
        }
        $c += "`"-tags`" " + "`"autogen`" "
        $c += "`"-check.timeout`" " + "`"10m`" "
        $c += "`"-test.timeout`" " + "`"200m`" "

        if ($env:INTEGRATION_IN_CONTAINER -ne $null) {
            Write-Host -ForegroundColor Green "INFO: Integration tests being run inside a container"
            # Note we talk back through the containers gateway address
            # And the ridiculous lengths we have to go to to get the default gateway address... (GetNetIPConfiguration doesn't work in nanoserver)
            # I just could not get the escaping to work in a single command, so output $c to a file and run that in the container instead...
            # Not the prettiest, but it works.
            $c | Out-File -Force "$env:TEMP\binary\runIntegrationCLI.ps1"
            $Duration= $(Measure-Command { & docker run `
                                                    --rm `
                                                    -e c=$c `
                                                    --workdir "c`:\go\src\github.com\docker\docker\integration-cli" `
                                                    -v "$env:TEMP\binary`:c:\target" `
                                                    docker `
                                                    "`$env`:PATH`='c`:\target;'+`$env:PATH`;  `$env:DOCKER_HOST`='tcp`://'+(ipconfig | select -last 1).Substring(39)+'`:2357'; c:\target\runIntegrationCLI.ps1" | Out-Host } )
        } else  {
            Write-Host -ForegroundColor Green "INFO: Integration tests being run from the host:"
            cd "$env:SOURCES_DRIVE`:\$env:SOURCES_SUBDIR\src\github.com\docker\docker\integration-cli"
            $env:DOCKER_HOST=$DASHH_CUT
            $env:PATH="$env:TEMP\binary;$env:PATH;"  # Force to use the test binaries, not the host ones.
            Write-Host -ForegroundColor Green "INFO: $c"
            Write-Host -ForegroundColor Green "INFO: DOCKER_HOST at $DASHH_CUT"
            # Explicit to not use measure-command otherwise don't get output as it goes
            $start=(Get-Date); Invoke-Expression $c; $Duration=New-Timespan -Start $start -End (Get-Date)
        }
        $ErrorActionPreference = "Stop"
        if (-not($LastExitCode -eq 0)) {
            Throw "ERROR: Integration tests failed at $(Get-Date). Duration`:$Duration"
        }
        Write-Host  -ForegroundColor Green "INFO: Integration tests ended at $(Get-Date). Duration`:$Duration"
    }else {
        Write-Host -ForegroundColor Magenta "WARN: Skipping integration tests"
    }

    # Docker info now to get counts (after or if jjh/containercounts is merged)
    if ($daemonStarted -eq 1) {
        Write-Host -ForegroundColor Green "INFO: Docker info of the daemon under test at end of run"
        Write-Host
        $ErrorActionPreference = "SilentlyContinue"
        & "$env:TEMP\binary\docker-$COMMITHASH" "-H=$($DASHH_CUT)" info
        $ErrorActionPreference = "Stop"
        if ($LastExitCode -ne 0) {
            Throw "ERROR: The daemon under test does not appear to be running."
            $DumpDaemonLog=1
        }
        Write-Host
    }

    # Stop the daemon under test
    if ($daemonStarted -eq 1) {
        if (Test-Path "$env:TEMP\docker.pid") {
            $p=Get-Content "$env:TEMP\docker.pid" -raw
            if ($p -ne $null) {
                Write-Host -ForegroundColor green "INFO: Stopping daemon under test"
                taskkill -f -t -pid $p
                Remove-Item "$env:TEMP\docker.pid" -force -ErrorAction SilentlyContinue
                #sleep 5
            }
        }
    }

    Write-Host -ForegroundColor Green "INFO: executeCI.ps1 Completed successfully at $(Get-Date)."
}
Catch [Exception] {
    $FinallyColour="Red"
    Write-Host -ForegroundColor Red ("`r`n`r`nERROR: Failed '$_' at $(Get-Date)")
    Write-Host "`n`n"

    # Exit to ensure Jenkins captures it. Don't do this in the ISE or interactive Powershell - they will catch the Throw onwards.
    if ( ([bool]([Environment]::GetCommandLineArgs() -Like '*-NonInteractive*')) -and `
        ([bool]([Environment]::GetCommandLineArgs() -NotLike "*Powershell_ISE.exe*"))) {
        exit 1
    }
    Throw $_
}
Finally {
    $ErrorActionPreference="SilentlyContinue"
    $global:ProgressPreference=$origProgressPreference
    Write-Host  -ForegroundColor Green "INFO: Tidying up at end of run"

    # Restore the path
    if ($origPath -ne $null) { $env:PATH=$origPath }

    # Restore the DOCKER_HOST
    if ($origDOCKER_HOST -ne $null) { $env:DOCKER_HOST=$origDOCKER_HOST }

    # Restore the GOROOT and GOPATH variables
    if ($origGOROOT -ne $null) { $env:GOROOT=$origGOROOT }
    if ($origGOPATH -ne $null) { $env:GOPATH=$origGOPATH }

    # Dump the daemon log if asked to
    if ($daemonStarted -eq 1) {
        if ($dumpDaemonLog -eq 1) {
            Write-Host -ForegroundColor Cyan "----------- DAEMON LOG ------------"
            Get-Content "$env:TEMP\dut.err" -ErrorAction SilentlyContinue | Write-Host -ForegroundColor Cyan
            Write-Host -ForegroundColor Cyan "----------- END DAEMON LOG --------"
        }
    }

    # Save the daemon under test log
    if ($daemonStarted -eq 1) {
        Write-Host -ForegroundColor Green "INFO: Saving daemon under test log ($env:TEMP\dut.err) to $TEMPORIG\CIDUT.log"
        Copy-Item  "$env:TEMP\dut.err" "$TEMPORIG\CIDUT.log" -Force -ErrorAction SilentlyContinue
    }

    cd "$env:SOURCES_DRIVE\$env:SOURCES_SUBDIR" -ErrorAction SilentlyContinue
    Nuke-Everything
    $Dur=New-TimeSpan -Start $StartTime -End $(Get-Date)
    Write-Host -ForegroundColor $FinallyColour "`nINFO: executeCI.ps1 exiting at $(date). Duration $dur`n"
}

