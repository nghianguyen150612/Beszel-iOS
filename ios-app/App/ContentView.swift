import SwiftUI

struct ContentView: View {
    @StateObject private var controller = AgentController()
    @Environment(\.scenePhase) private var scenePhase
    var body: some View {
        NavigationStack {
            Form {
                Section("Connection") {
                    TextField("HTTPS Hub URL", text: $controller.hubURL)
                        .textInputAutocapitalization(.never).autocorrectionDisabled()
                    TextField("Hub public key", text: $controller.hubKey)
                        .textInputAutocapitalization(.never).autocorrectionDisabled()
                    SecureField("Registration token", text: $controller.token)
                    Button("Save configuration") { _ = controller.save() }
                    Button("Connect") { controller.connect() }
                    Button("Disconnect") { controller.disconnect() }
                    LabeledContent("Status", value: controller.state)
                    if !controller.error.isEmpty { Text(controller.error).foregroundStyle(.red) }
                }
                Section("Diagnostics") {
                    LabeledContent("Beszel", value: goString(BeszelGoVersion()))
                    LabeledContent("Go target", value: goString(BeszelGoPlatform()))
                    LabeledContent("Device", value: StockMetrics.modelIdentifier() ?? "Unavailable")
                    Text(controller.diagnostics).font(.caption).textSelection(.enabled)
                }
                Section {
                    Text("Foreground only. Reconnect explicitly after returning to the app. Metric probes appear in local diagnostics. Hub stats require a complete system snapshot, including whole-system storage and network measurements.")
                        .font(.footnote)
                }
            }.navigationTitle("Beszel Agent")
        }.onChange(of: scenePhase) { phase in controller.sceneChanged(phase) }
    }
}
