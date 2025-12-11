# Parse benchmark results
$content = Get-Content all_benchmarks.txt
$benchmarks = @{}
$currentBench = ""

foreach($line in $content) {
    # Match benchmark name (may have debug output after it)
    if($line -match "^Benchmark(\w+)-8\s+") {
        $currentBench = $matches[1]
    }
    # Match performance metrics
    elseif($currentBench -and $line -match "^\s+(\d+)\s+([\d\.]+)\s+ns/op\s+(\d+)\s+B/op\s+(\d+)\s+allocs/op") {
        $benchmarks[$currentBench] = @{
            ops = $matches[1]
            ns = $matches[2]
            bytes = $matches[3]
            allocs = $matches[4]
        }
        $currentBench = ""
    }
}

# Output formatted results
$results = @()
$benchmarks.Keys | Sort-Object | ForEach-Object {
    $b = $benchmarks[$_]
    $results += "{0,-35} {1,10} ops  {2,8} ns/op  {3,5} B/op  {4,2} allocs/op" -f $_, $b.ops, $b.ns, $b.bytes, $b.allocs
}

$results | Out-File benchmark_results_summary.txt
Write-Host "Parsed $($benchmarks.Count) benchmarks"
$results
