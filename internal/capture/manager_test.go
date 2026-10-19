package capture

import "testing"

func TestFirstPID(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{name: "single", output: "1234\n", want: "1234"},
		{name: "multiple", output: "4321 9876\n", want: "4321"},
		{name: "ignores invalid", output: "screenrecord -1 0 2468", want: "2468"},
		{name: "empty", output: "", want: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := firstPID(test.output); got != test.want {
				t.Fatalf("firstPID(%q) = %q, want %q", test.output, got, test.want)
			}
		})
	}
}

func TestShellQuote(t *testing.T) {
	got := shellQuote("yokasgs://?payload=1|24,example's$(command).sgs")
	want := "'yokasgs://?payload=1|24,example'\\''s$(command).sgs'"
	if got != want {
		t.Fatalf("shellQuote() = %q, want %q", got, want)
	}
}

func TestParseGameClients(t *testing.T) {
	output := `2 activities found:
  Activity #0:
    com.bf.sgs.hdexp/org.cocos2dx.lua.AppActivity
  Activity #1:
    com.bf.sgs.hdexp.huawei/org.cocos2dx.lua.AppActivity
`
	clients := parseGameClients(output)
	if len(clients) != 2 {
		t.Fatalf("len(parseGameClients()) = %d, want 2", len(clients))
	}
	if clients[1].PackageName != "com.bf.sgs.hdexp.huawei" {
		t.Fatalf("second package = %q", clients[1].PackageName)
	}
}

func TestSelectGameClient(t *testing.T) {
	clients := parseGameClients("com.bf.sgs.hdexp/Main\ncom.bf.sgs.hdexp.huawei/Main\n")
	if _, err := selectGameClient(clients, ""); err == nil {
		t.Fatal("selectGameClient() accepted an ambiguous client list")
	}
	got, err := selectGameClient(clients, "com.bf.sgs.hdexp.huawei")
	if err != nil {
		t.Fatalf("selectGameClient() error = %v", err)
	}
	if got != "com.bf.sgs.hdexp.huawei" {
		t.Fatalf("selectGameClient() = %q", got)
	}
}
