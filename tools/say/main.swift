import CoreAudio
import Darwin

let systemSay = "/usr/bin/say"
let silentOptions = ["-o", "-n", "--output-file", "--network-send"]

func values(_ object: AudioObjectID, _ selector: AudioObjectPropertySelector) -> [UInt32] {
    var address = AudioObjectPropertyAddress(mSelector: selector, mScope: kAudioObjectPropertyScopeGlobal, mElement: kAudioObjectPropertyElementMain)
    var size: UInt32 = 0
    guard AudioObjectGetPropertyDataSize(object, &address, 0, nil, &size) == noErr else { return [] }
    var result = [UInt32](repeating: 0, count: Int(size) / MemoryLayout<UInt32>.size)
    guard AudioObjectGetPropertyData(object, &address, 0, nil, &size, &result) == noErr else { return [] }
    return result
}

func micInUse() -> Bool {
    let processes = values(AudioObjectID(kAudioObjectSystemObject), kAudioHardwarePropertyProcessObjectList)
    return processes.contains { values($0, kAudioProcessPropertyIsRunningInput) == [1] }
}

func speaksAloud(_ arguments: ArraySlice<String>) -> Bool {
    for argument in arguments {
        if argument == "--" { return true }
        if argument == "?" || argument.hasSuffix("=?") { return false }
        if silentOptions.contains(where: { argument.hasPrefix($0) }) { return false }
    }
    return true
}

if speaksAloud(CommandLine.arguments.dropFirst()) && micInUse() { exit(0) }
execv(systemSay, CommandLine.unsafeArgv)
perror(systemSay)
exit(1)
