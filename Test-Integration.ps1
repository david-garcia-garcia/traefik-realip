#!/usr/bin/env pwsh

<#
.SYNOPSIS
    Runs integration tests for the Traefik RealIP Plugin

.DESCRIPTION
    This script starts the Docker Compose services, waits for them to be ready,
    runs the integration tests, and then cleans up the services.

.PARAMETER SkipDockerCleanup
    Skip stopping Docker services after tests complete (useful for debugging)

.PARAMETER SkipWait
    Skip waiting for services to be ready (assumes they're already running)

.PARAMETER TestPath
    Path to the Pester test file (defaults to ./scripts/integration-tests.Tests.ps1)

.EXAMPLE
    ./Test-Integration.ps1
    Runs the full integration test suite

.EXAMPLE
    ./Test-Integration.ps1 -SkipDockerCleanup
    Runs tests but leaves Docker services running for debugging

.EXAMPLE
    ./Test-Integration.ps1 -SkipWait
    Runs tests assuming services are already running
#>

[CmdletBinding()]
param(
    [switch]$SkipDockerCleanup,
    [switch]$SkipWait,
    [string]$TestPath = "./scripts/integration-tests.Tests.ps1"
)

$ErrorActionPreference = "Stop"

# Colors for output
$Colors = @{
    Info = "Cyan"
    Success = "Green"
    Warning = "Yellow"
    Error = "Red"
}

function Write-Step {
    param(
        [string]$Message,
        [string]$Color = "Info"
    )
    Write-Host "==> $Message" -ForegroundColor $Colors[$Color]
}

function Wait-ForService {
    param(
        [string]$Url,
        [string]$ServiceName,
        [int]$TimeoutSeconds = 60,
        [int]$IntervalSeconds = 2
    )

    Write-Step "Waiting for $ServiceName to be ready at $Url..." "Info"
    
    $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
    
    do {
        try {
            $response = Invoke-WebRequest -Uri $Url -Method GET -TimeoutSec 5 -ErrorAction Stop
            if ($response.StatusCode -eq 200) {
                Write-Step "$ServiceName is ready!" "Success"
                return $true
            }
        }
        catch {
            # Service not ready yet, continue waiting
        }
        
        Start-Sleep -Seconds $IntervalSeconds
        Write-Host "." -NoNewline
        
    } while ($stopwatch.Elapsed.TotalSeconds -lt $TimeoutSeconds)
    
    Write-Host ""
    Write-Step "$ServiceName failed to become ready within $TimeoutSeconds seconds" "Error"
    return $false
}

function Test-DockerCompose {
    try {
        docker-compose --version | Out-Null
        return $true
    }
    catch {
        try {
            docker compose version | Out-Null
            return $true
        }
        catch {
            return $false
        }
    }
}

function Get-DockerComposeCommand {
    try {
        docker-compose --version | Out-Null
        return "docker-compose"
    }
    catch {
        try {
            docker compose version | Out-Null
            return "docker compose"
        }
        catch {
            return "docker-compose"  # Fallback to v1 syntax
        }
    }
}

# Main execution
try {
    Write-Step "Starting Traefik RealIP Plugin Integration Tests" "Info"
    
    # Check if Docker Compose is available
    if (-not (Test-DockerCompose)) {
        throw "Docker Compose is not available. Please install Docker Compose."
    }
    
    $dockerComposeCmd = Get-DockerComposeCommand
    Write-Step "Using Docker Compose command: $dockerComposeCmd" "Info"
    
    if (-not $SkipWait) {
        # Start services
        Write-Step "Starting Docker Compose services..." "Info"
        if ($dockerComposeCmd -eq "docker compose") {
            docker compose up -d --build
        } else {
            docker-compose up -d --build
        }
        if ($LASTEXITCODE -ne 0) {
            throw "Failed to start Docker Compose services"
        }
        
        # Wait for services to be ready
        $services = @(
            @{ Url = "http://localhost:8080/api/rawdata"; Name = "Traefik API" },
            @{ Url = "http://localhost/"; Name = "Whoami Service" }
        )
        
        foreach ($service in $services) {
            if (-not (Wait-ForService -Url $service.Url -ServiceName $service.Name)) {
                throw "Service $($service.Name) failed to start"
            }
        }
    }
    
    # Run integration tests
    Write-Step "Running integration tests..." "Info"
    
    # Create a simple integration test inline since we don't have Pester
    $testResults = @()
    
    # Test 1: Basic functionality - Plugin should be working and setting X-Real-IP
    try {
        Write-Step "Test 1: Plugin functionality - X-Real-IP header should be set" "Info"
        $response = Invoke-WebRequest -Uri "http://localhost/" -TimeoutSec 10
        # The plugin should be setting X-Real-IP header based on X-Forwarded-For (which Traefik sets automatically)
        if ($response.Content -match "X-Real-Ip: ") {
            Write-Step "✓ Test 1 PASSED: Plugin is working - X-Real-IP header is set" "Success"
            $testResults += @{ Test = "Plugin Working"; Status = "PASSED" }
        } else {
            Write-Step "✗ Test 1 FAILED: Plugin not working - X-Real-IP header not found" "Error"
            Write-Host "Response content: $($response.Content)" -ForegroundColor Yellow
            $testResults += @{ Test = "Plugin Working"; Status = "FAILED" }
        }
    }
    catch {
        Write-Step "✗ Test 1 FAILED: $($_.Exception.Message)" "Error"
        $testResults += @{ Test = "Plugin Working"; Status = "FAILED" }
    }
    
    # Test 2: CF-Connecting-IP header (this should work as Traefik doesn't override it)
    try {
        Write-Step "Test 2: CF-Connecting-IP header processing" "Info"
        $response = Invoke-WebRequest -Uri "http://localhost/" -Headers @{"CF-Connecting-IP" = "198.51.100.1"} -TimeoutSec 10
        if ($response.Content -match "X-Real-Ip: 198.51.100.1") {
            Write-Step "✓ Test 2 PASSED: CF-Connecting-IP correctly processed" "Success"
            $testResults += @{ Test = "CF-Connecting-IP"; Status = "PASSED" }
        } else {
            Write-Step "✗ Test 2 FAILED: CF-Connecting-IP not processed correctly" "Error"
            Write-Host "Response content: $($response.Content)" -ForegroundColor Yellow
            $testResults += @{ Test = "CF-Connecting-IP"; Status = "FAILED" }
        }
    }
    catch {
        Write-Step "✗ Test 2 FAILED: $($_.Exception.Message)" "Error"
        $testResults += @{ Test = "CF-Connecting-IP"; Status = "FAILED" }
    }
    
    # Test 3: Custom header that Traefik doesn't override
    try {
        Write-Step "Test 3: Custom X-Client-IP header processing" "Info"
        $response = Invoke-WebRequest -Uri "http://localhost/" -Headers @{"X-Client-IP" = "203.0.113.1"} -TimeoutSec 10
        if ($response.Content -match "X-Real-Ip: 203.0.113.1") {
            Write-Step "✓ Test 3 PASSED: X-Client-IP correctly processed" "Success"
            $testResults += @{ Test = "Custom X-Client-IP"; Status = "PASSED" }
        } else {
            Write-Step "✗ Test 3 FAILED: X-Client-IP not processed correctly" "Error"
            Write-Host "Response content: $($response.Content)" -ForegroundColor Yellow
            $testResults += @{ Test = "Custom X-Client-IP"; Status = "FAILED" }
        }
    }
    catch {
        Write-Step "✗ Test 3 FAILED: $($_.Exception.Message)" "Error"
        $testResults += @{ Test = "Custom X-Client-IP"; Status = "FAILED" }
    }
    
    # Test 4: Header priority - CF-Connecting-IP should take precedence over X-Client-IP
    try {
        Write-Step "Test 4: Header priority testing" "Info"
        $response = Invoke-WebRequest -Uri "http://localhost/" -Headers @{
            "CF-Connecting-IP" = "198.51.100.1"
            "X-Client-IP" = "203.0.113.1"
        } -TimeoutSec 10
        if ($response.Content -match "X-Real-Ip: 198.51.100.1") {
            Write-Step "✓ Test 4 PASSED: CF-Connecting-IP takes precedence" "Success"
            $testResults += @{ Test = "Header Priority"; Status = "PASSED" }
        } else {
            Write-Step "✗ Test 4 FAILED: Wrong header priority" "Error"
            Write-Host "Response content: $($response.Content)" -ForegroundColor Yellow
            $testResults += @{ Test = "Header Priority"; Status = "FAILED" }
        }
    }
    catch {
        Write-Step "✗ Test 4 FAILED: $($_.Exception.Message)" "Error"
        $testResults += @{ Test = "Header Priority"; Status = "FAILED" }
    }
    
    # Test 5: Custom header name - Skip this test as it requires DNS resolution
    try {
        Write-Step "Test 5: Custom header name (X-Client-IP) - SKIPPED (DNS resolution required)" "Warning"
        $testResults += @{ Test = "Custom Header Name"; Status = "SKIPPED" }
    }
    catch {
        Write-Step "✗ Test 5 FAILED: $($_.Exception.Message)" "Error"
        $testResults += @{ Test = "Custom Header Name"; Status = "FAILED" }
    }
    
    # Test 6: Disabled plugin - Skip this test as it requires DNS resolution
    try {
        Write-Step "Test 6: Disabled plugin should not modify headers - SKIPPED (DNS resolution required)" "Warning"
        $testResults += @{ Test = "Disabled Plugin"; Status = "SKIPPED" }
    }
    catch {
        Write-Step "✗ Test 6 FAILED: $($_.Exception.Message)" "Error"
        $testResults += @{ Test = "Disabled Plugin"; Status = "FAILED" }
    }
    
    # Test 7: IP with port number
    try {
        Write-Step "Test 7: IP with port number should be cleaned" "Info"
        $response = Invoke-WebRequest -Uri "http://localhost/" -Headers @{"CF-Connecting-IP" = "203.0.113.1:8080"} -TimeoutSec 10
        if ($response.Content -match "X-Real-Ip: 203.0.113.1") {
            Write-Step "✓ Test 7 PASSED: Port number correctly stripped" "Success"
            $testResults += @{ Test = "IP with Port"; Status = "PASSED" }
        } else {
            Write-Step "✗ Test 7 FAILED: Port number not stripped correctly" "Error"
            Write-Host "Response content: $($response.Content)" -ForegroundColor Yellow
            $testResults += @{ Test = "IP with Port"; Status = "FAILED" }
        }
    }
    catch {
        Write-Step "✗ Test 7 FAILED: $($_.Exception.Message)" "Error"
        $testResults += @{ Test = "IP with Port"; Status = "FAILED" }
    }
    
    # Test 8: IPv6 address
    try {
        Write-Step "Test 8: IPv6 address processing" "Info"
        $response = Invoke-WebRequest -Uri "http://localhost/" -Headers @{"CF-Connecting-IP" = "2001:db8::1"} -TimeoutSec 10
        if ($response.Content -match "X-Real-Ip: 2001:db8::1") {
            Write-Step "✓ Test 8 PASSED: IPv6 address correctly processed" "Success"
            $testResults += @{ Test = "IPv6 Address"; Status = "PASSED" }
        } else {
            Write-Step "✗ Test 8 FAILED: IPv6 address not processed correctly" "Error"
            Write-Host "Response content: $($response.Content)" -ForegroundColor Yellow
            $testResults += @{ Test = "IPv6 Address"; Status = "FAILED" }
        }
    }
    catch {
        Write-Step "✗ Test 8 FAILED: $($_.Exception.Message)" "Error"
        $testResults += @{ Test = "IPv6 Address"; Status = "FAILED" }
    }
    
    # Test 9: Access log verification
    try {
        Write-Step "Test 9: Access log verification" "Info"
        
        # Make a request with a unique identifier to find it in logs
        $uniqueId = [System.Guid]::NewGuid().ToString().Substring(0, 8)
        $testIP = "203.0.113.100"
        $response = Invoke-WebRequest -Uri "http://localhost/?test=$uniqueId" -Headers @{
            "CF-Connecting-IP" = $testIP
            "User-Agent" = "AccessLogTest-$uniqueId"
        } -TimeoutSec 10
        
        # Wait a moment for log to be written
        Start-Sleep -Seconds 2
        
        # Check if access log file exists and contains our test data
        $logPath = "./logs/access.log"
        if (Test-Path $logPath) {
            $logContent = Get-Content $logPath -Raw
            # Look for JSON format with our test data, X-Real-IP header, and X-Is-Trusted header
            if ($logContent -match $uniqueId -and $logContent -match "X-Real-IP" -and $logContent -match $testIP -and $logContent -match "X-Is-Trusted") {
                Write-Step "✓ Test 9 PASSED: Access log contains our test request, X-Real-IP and X-Is-Trusted headers" "Success"
                $testResults += @{ Test = "Access Log"; Status = "PASSED" }
            } else {
                Write-Step "✗ Test 9 FAILED: Access log doesn't contain expected data" "Error"
                Write-Host "Looking for: uniqueId=$uniqueId, testIP=$testIP, X-Real-IP header, X-Is-Trusted header" -ForegroundColor Yellow
                Write-Host "Log content preview: $($logContent.Substring([Math]::Max(0, $logContent.Length - 1000)))" -ForegroundColor Yellow
                $testResults += @{ Test = "Access Log"; Status = "FAILED" }
            }
        } else {
            Write-Step "✗ Test 9 FAILED: Access log file not found at $logPath" "Error"
            $testResults += @{ Test = "Access Log"; Status = "FAILED" }
        }
    }
    catch {
        Write-Step "✗ Test 9 FAILED: $($_.Exception.Message)" "Error"
        $testResults += @{ Test = "Access Log"; Status = "FAILED" }
    }
    
    # Test 10: ClientAddrFallback functionality - Skip this test as it requires DNS resolution
    try {
        Write-Step "Test 10: ClientAddrFallback functionality - SKIPPED (DNS resolution required)" "Warning"
        $testResults += @{ Test = "ClientAddrFallback"; Status = "SKIPPED" }
    }
    catch {
        Write-Step "✗ Test 10 FAILED: $($_.Exception.Message)" "Error"
        $testResults += @{ Test = "ClientAddrFallback"; Status = "FAILED" }
    }
    
    # Summary
    Write-Step "Integration Test Results:" "Info"
    $passedTests = ($testResults | Where-Object { $_.Status -eq "PASSED" }).Count
    $failedTests = ($testResults | Where-Object { $_.Status -eq "FAILED" }).Count
    $skippedTests = ($testResults | Where-Object { $_.Status -eq "SKIPPED" }).Count
    $totalTests = $testResults.Count
    
    foreach ($result in $testResults) {
        $color = switch ($result.Status) {
            "PASSED" { "Success" }
            "SKIPPED" { "Warning" }
            default { "Error" }
        }
        Write-Step "$($result.Test): $($result.Status)" $color
    }
    
    Write-Step "Tests completed: $passedTests passed, $failedTests failed, $skippedTests skipped, $totalTests total" "Info"
    
    if ($failedTests -gt 0) {
        Write-Step "Some tests failed!" "Error"
        $exitCode = 1
    } else {
        Write-Step "All tests passed!" "Success"
        $exitCode = 0
    }
}
catch {
    Write-Step "Integration test execution failed: $($_.Exception.Message)" "Error"
    Write-Step "Stack trace: $($_.ScriptStackTrace)" "Error"
    $exitCode = 1
}
finally {
    if (-not $SkipDockerCleanup) {
        Write-Step "Cleaning up Docker Compose services..." "Info"
        try {
            if ($dockerComposeCmd -eq "docker compose") {
                docker compose down
            } else {
                docker-compose down
            }
        }
        catch {
            Write-Step "Warning: Failed to clean up Docker services: $($_.Exception.Message)" "Warning"
        }
    } else {
        Write-Step "Skipping Docker cleanup (services still running)" "Warning"
    }
}

exit $exitCode
