#requires -Version 2 -Modules posh-git

function Write-Theme {
    param(
        [bool]
        $lastCommandFailed,
        [string]
        $with
    )

    #check the last command state and indicate if failed
    $promtSymbolColor = $sl.Colors.PromptSymbolColor
    If ($lastCommandFailed) {
        $promtSymbolColor = $sl.Colors.WithForegroundColor
    }

    # Writes the postfixes to the prompt
    $prompt += Write-Prompt -Object ($sl.PromptSymbols.PromptIndicator + "  ") -ForegroundColor $promtSymbolColor

    # Writes the drive portion
    if ( $sl.PromptControl.DirPrompt ) {
        $drive = $sl.PromptSymbols.HomeSymbol
        if ($pwd.Path -ne $HOME) {
            $drive = "$(Split-Path -path $pwd -Leaf)"
        }
        $prompt += Write-Prompt -Object $drive -ForegroundColor $sl.Colors.DriveForegroundColor
    }

    if ( $sl.PromptControl.GitPrompt ) {
        $status = Get-VCSStatus
        if ($status) {
            if ( $sl.PromptControl.DirPrompt ) {
                $prompt +=  Write-Prompt -Object ' '
            }
            $prompt += Write-Prompt -Object "git:(" -ForegroundColor $sl.Colors.PromptHighlightColor
            $prompt += Write-Prompt -Object "$($status.Branch)" -ForegroundColor $sl.Colors.WithForegroundColor
            $prompt += Write-Prompt -Object ")" -ForegroundColor $sl.Colors.PromptHighlightColor
            if ($status.Working.Length -gt 0) {
                $prompt += Write-Prompt -Object (" " + $sl.PromptSymbols.GitDirtyIndicator) -ForegroundColor $sl.Colors.GitDefaultColor
            }
        }
    }

    $prompt += ' '
    $prompt
}

$sl = $global:ThemeSettings #local settings
$sl.PromptSymbols.PromptIndicator = [char]::ConvertFromUtf32(0x279C)
$sl.Colors.PromptSymbolColor = [ConsoleColor]::Green
$sl.Colors.PromptHighlightColor = [ConsoleColor]::Blue
$sl.Colors.DriveForegroundColor = [ConsoleColor]::Cyan
$sl.Colors.WithForegroundColor = [ConsoleColor]::Red
$sl.PromptSymbols.GitDirtyIndicator = [char]::ConvertFromUtf32(10007)
$sl.Colors.GitDefaultColor =[ConsoleColor]::Yellow
