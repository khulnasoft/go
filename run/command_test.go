package run_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/khulnasoft/go/run"
)

var outputTests = []func(c *qt.C, out run.Output, expect string, expectError bool){
	func(c *qt.C, out run.Output, expect string, expectError bool) {
		c.Run("Stream", func(c *qt.C) {
			var b bytes.Buffer
			err := out.Stream(&b)
			c.Assert(err, qt.IsNil)
			c.Assert(b.String(), qt.Equals, fmt.Sprintf("%s\n", expect))
		})
	},
	func(c *qt.C, out run.Output, expect string, expectError bool) {
		c.Run("StreamLines", func(c *qt.C) {
			linesC := make(chan string, 10)
			err := out.StreamLines(func(line string) {
				linesC <- line
			})
			c.Assert(err, qt.IsNil)
			close(linesC)

			var lines []string
			for l := range linesC {
				lines = append(lines, l)
			}
			c.Assert(len(lines), qt.Equals, 1)
			c.Assert(string(lines[0]), qt.Equals, expect)
		})
	},
	func(c *qt.C, out run.Output, expect string, expectError bool) {
		c.Run("Lines", func(c *qt.C) {
			lines, err := out.Lines()
			if !expectError {
				c.Assert(err, qt.IsNil)
			}
			c.Assert(len(lines), qt.Equals, 1)
			c.Assert(lines[0], qt.Equals, expect)
		})
	},
	func(c *qt.C, out run.Output, expect string, expectError bool) {
		c.Run("String", func(c *qt.C) {
			str, err := out.String()
			if !expectError {
				c.Assert(err, qt.IsNil)
			}
			c.Assert(str, qt.Equals, expect)
		})
	},
	func(c *qt.C, out run.Output, expect string, expectError bool) {
		c.Run("Read: fixed bytes", func(c *qt.C) {
			b := make([]byte, 100)
			n, err := out.Read(b)
			// We expect io.EOF if we are done reading - the content assertion
			// should still pass.
			if !expectError && err != io.EOF {
				c.Assert(err, qt.IsNil)
			}
			c.Assert(string(b[0:n]), qt.Equals, fmt.Sprintf("%s\n", expect))
		})
	},
	func(c *qt.C, out run.Output, expect string, expectError bool) {
		if expectError {
			return // not applicable
		}

		c.Run("Read: exactly length of output", func(c *qt.C) {
			// Read exactly the amount of output
			b := make([]byte, len(expect)+1)
			n, err := out.Read(b)
			c.Assert(err, qt.IsNil)
			c.Assert(string(b[0:n]), qt.Equals, fmt.Sprintf("%s\n", expect))

			// A subsequent read should indicate nothing read, and an EOF
			n, err = out.Read(make([]byte, 100))
			c.Assert(n, qt.Equals, 0)
			c.Assert(err, qt.Equals, io.EOF)
		})
	},
	func(c *qt.C, out run.Output, expect string, expectError bool) {
		c.Run("Read: io.ReadAll", func(c *qt.C) {
			b, err := io.ReadAll(out)
			if !expectError {
				c.Assert(err, qt.IsNil)
			}
			c.Assert(string(b), qt.Equals, fmt.Sprintf("%s\n", expect))
		})
	},
	func(c *qt.C, out run.Output, expect string, expectError bool) {
		c.Run("Wait", func(c *qt.C) {
			err := out.Wait()
			if !expectError {
				c.Assert(err, qt.IsNil)
			}
		})
	},
}

func TestRunAndAggregate(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()

	command := `echo "hello world"`

	type testCase struct {
		name        string
		output      func() run.Output
		expect      string
		expectError bool
	}
	for _, tc := range []testCase{
		{
			name: "plain output",
			output: func() run.Output {
				return run.Cmd(ctx, command).Run()
			},
			expect: "hello world",
		},
		{
			name: "plain output and exit with error",
			output: func() run.Output {
				return run.Cmd(ctx, command, "; exit 1").Run()
			},
			expect:      "hello world ; exit 1",
			expectError: true,
		},
		{
			name: "mapped output",
			output: func() run.Output {
				return run.Cmd(ctx, command).Run().
					Map(func(ctx context.Context, line []byte, dst io.Writer) (int, error) {
						return dst.Write(bytes.ReplaceAll(line, []byte("hello"), []byte("goodbye")))
					})
			},
			expect:      "goodbye world",
			expectError: true, // io.EOF
		},
		{
			name: "multiple mapped output",
			output: func() run.Output {
				return run.Cmd(ctx, command).Run().
					Map(func(ctx context.Context, line []byte, dst io.Writer) (int, error) {
						return dst.Write(bytes.ReplaceAll(line, []byte("hello"), []byte("goodbye")))
					}).
					Map(func(ctx context.Context, line []byte, dst io.Writer) (int, error) {
						return dst.Write(bytes.ReplaceAll(line, []byte("world"), []byte("jh")))
					})
			},
			expect:      "goodbye jh",
			expectError: true, // io.EOF
		},
	} {
		c.Run(tc.name, func(c *qt.C) {
			for _, test := range outputTests {
				test(c, tc.output(), tc.expect, tc.expectError)
			}
		})
	}
}

func TestJQ(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()

	c.Run("cat and JQ", func(c *qt.C) {
		const testJSON = `{
			"hello": "world"
		}`

		res, err := run.Cmd(ctx, "cat").
			Input(strings.NewReader(testJSON)).
			Run().
			JQ(".hello")
		c.Assert(err, qt.IsNil)
		c.Assert(string(res), qt.Equals, `"world"`)
	})

}

func TestEdgeCases(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()

	c.Run("empty lines from map are preserved", func(c *qt.C) {
		const testData = `hello

		world`

		c.Run("without map", func(c *qt.C) {
			res, err := run.Cmd(ctx, "cat").
				Input(strings.NewReader(testData)).
				Run().
				Lines()
			c.Assert(err, qt.IsNil)
			c.Assert(len(res), qt.Equals, 3)
		})

		c.Run("with map", func(c *qt.C) {
			res, err := run.Cmd(ctx, "cat").
				Input(strings.NewReader(testData)).
				Run().
				Map(func(ctx context.Context, line []byte, dst io.Writer) (int, error) {
					return dst.Write(line)
				}).
				Lines()
			c.Assert(err, qt.IsNil)
			c.Assert(len(res), qt.Equals, 3)
		})
	})

	c.Run("mixed output", func(c *qt.C) {
		const mixedOutputCmd = `echo "stdout" ; sleep 0.001 ; >&2 echo "stderr"`

		c.Run("stdout only", func(c *qt.C) {
			res, err := run.Bash(ctx, mixedOutputCmd).
				StdOut().
				Run().
				Lines()
			c.Assert(err, qt.IsNil)
			c.Assert(res, qt.CmpEquals(), []string{"stdout"})
		})

		c.Run("stderr only", func(c *qt.C) {
			res, err := run.Bash(ctx, mixedOutputCmd).
				StdErr().
				Run().
				Lines()
			c.Assert(err, qt.IsNil)
			c.Assert(res, qt.CmpEquals(), []string{"stderr"})
		})

		c.Run("combined", func(c *qt.C) {
			res, err := run.Bash(ctx, mixedOutputCmd).
				Run().
				Lines()
			c.Assert(err, qt.IsNil)
			c.Assert(res, qt.CmpEquals(), []string{"stdout", "stderr"})
		})
	})
}

func TestBashOpts(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()

	c.Run("depending on bash mode, a pipe command that fails should return an exit code", func(c *qt.C) {
		pipeCmd := "echo '123456789' | grep 999 | echo 1"
		c.Run("normal bash -c - pipe that fails should not exit with non zero command", func(c *qt.C) {
			// In the pipe grep 999 will fail, but since the last command in the pipe is a success the entire command succeeds
			_, err := run.Bash(ctx, pipeCmd).StdOut().Run().String()
			c.Assert(err, qt.IsNil)
		})
		c.Run("pipe command should fail with StrictBash", func(c *qt.C) {
			// with StrictBashOpts, since 'grep 999' fails in the pipe, the entire command is considered to have failed
			_, err := run.BashWith(ctx, run.StrictBashOpts, pipeCmd).StdOut().Run().String()
			c.Assert(err, qt.IsNotNil)
		})

	})
}

func TestInput(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()

	c.Run("set multiple inputs", func(c *qt.C) {
		cmd := run.Cmd(ctx, "cat").
			Input(strings.NewReader("hello")).
			Input(strings.NewReader(" ")).
			Input(strings.NewReader("world\n"))

		lines, err := cmd.Run().Lines()
		c.Assert(err, qt.IsNil)
		c.Assert(lines, qt.CmpEquals(), []string{"hello world"})
	})

	c.Run("reset input", func(c *qt.C) {
		cmd := run.Cmd(ctx, "cat").
			Input(strings.NewReader("hello")).
			ResetInput().
			Input(strings.NewReader("world"))

		lines, err := cmd.Run().Lines()
		c.Assert(err, qt.IsNil)
		c.Assert(lines, qt.CmpEquals(), []string{"world"})
	})
}
