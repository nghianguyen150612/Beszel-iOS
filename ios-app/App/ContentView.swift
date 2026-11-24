import SwiftUI

// P1 proves that Swift calls an actual linked Go runtime and shared Beszel
// version module. It does not simulate a running Agent or a connected Hub.
struct ContentView: View {
    private let beszelVersion = goString(BeszelGoVersion())
    private let goPlatform = goString(BeszelGoPlatform())

    var body: some View {
        NavigationStack {
            Form {
                Section("Native Go bridge") {
                    LabeledContent("Beszel version", value: beszelVersion)
                    LabeledContent("Go target", value: goPlatform)
                }
                Section("Stock iOS Agent") {
                    LabeledContent("Agent", value: "Not started (P2)")
                    LabeledContent("Hub", value: "Not configured (P2)")
                    LabeledContent("Metrics", value: "Not collected (P2)")
                }
                Section {
                    Text("P1 verifies a statically linked Go component. Foreground Agent transport, sandbox-compatible metrics and lifecycle handling are planned for P2/P3. No background daemon is claimed.")
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                }
            }
            .navigationTitle("Beszel Agent")
        }
    }
}

private func goString(_ pointer: UnsafeMutablePointer<CChar>?) -> String {
    guard let pointer else { return "Native bridge unavailable" }
    defer { BeszelGoFree(pointer) }
    return String(cString: pointer)
}
