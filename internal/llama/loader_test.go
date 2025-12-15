package llama

import "testing"

func TestMissingKnownDependency(t *testing.T) {
	ext := ".so"

	cases := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "detect cuda",
			err:  fakeErr("dlopen /home/user/bin/libggml.so: libggml-cuda.so: cannot open shared object file: No such file or directory"),
			want: "libggml-cuda.so",
		},
		{
			name: "detect metal",
			err:  fakeErr("dlopen /opt/libggml.so: libggml-metal.so: image not found"),
			want: "libggml-metal.so",
		},
		{
			name: "ignore unrelated",
			err:  fakeErr("dlopen libsomething.so: libdoesnotexist.so: cannot open shared object file"),
			want: "",
		},
		{
			name: "nil error",
			err:  nil,
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := missingKnownDependency(tc.err, ext)
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

type fakeErr string

func (e fakeErr) Error() string { return string(e) }
