package telnet_test

import (
	"net"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"euphio/internal/app"
	"euphio/internal/network/telnet"
)

var _ = Describe("Telnet Protocol", func() {
	var (
		serverConn net.Conn
		clientConn net.Conn
		connection *telnet.Connection
	)

	BeforeEach(func() {
		serverConn, clientConn = net.Pipe()
		connection = telnet.NewConnection(serverConn, app.Logger)

		// Set deadlines to prevent infinite hangs
		serverConn.SetDeadline(time.Now().Add(2 * time.Second))
		clientConn.SetDeadline(time.Now().Add(2 * time.Second))
	})

	AfterEach(func() {
		connection.Close()
		clientConn.Close()
	})

	Context("Negotiation", func() {
		It("should respond to DO ECHO with WILL ECHO", func() {
			// Start reading on the server side to process incoming data
			go func() {
				defer GinkgoRecover()
				buf := make([]byte, 1024)
				// Keep reading until error (EOF or closed)
				for {
					_, err := connection.Read(buf)
					if err != nil {
						return
					}
				}
			}()

			// Client sends DO ECHO
			_, err := clientConn.Write([]byte{telnet.IAC, telnet.DO, telnet.Echo})
			Expect(err).NotTo(HaveOccurred())

			// Check what server sends back
			buf := make([]byte, 1024)
			n, err := clientConn.Read(buf)
			Expect(err).NotTo(HaveOccurred())

			// Expect IAC WILL ECHO
			expected := []byte{telnet.IAC, telnet.WILL, telnet.Echo}
			Expect(buf[:n]).To(Equal(expected))

			// Verify internal state
			Eventually(func() bool {
				return connection.IsLocalOptionEnabled(telnet.Echo)
			}).Should(BeTrue())
		})

		It("should respond to WILL NAWS with DO NAWS", func() {
			go func() {
				defer GinkgoRecover()
				buf := make([]byte, 1024)
				for {
					_, err := connection.Read(buf)
					if err != nil {
						return
					}
				}
			}()

			// Client sends WILL NAWS
			_, err := clientConn.Write([]byte{telnet.IAC, telnet.WILL, telnet.NAWS})
			Expect(err).NotTo(HaveOccurred())

			buf := make([]byte, 1024)
			n, err := clientConn.Read(buf)
			Expect(err).NotTo(HaveOccurred())

			// Expect IAC DO NAWS
			expected := []byte{telnet.IAC, telnet.DO, telnet.NAWS}
			Expect(buf[:n]).To(Equal(expected))

			Eventually(func() bool {
				return connection.IsRemoteOptionEnabled(telnet.NAWS)
			}).Should(BeTrue())
		})

		It("should handle AYT command", func() {
			go func() {
				defer GinkgoRecover()
				buf := make([]byte, 1024)
				for {
					_, err := connection.Read(buf)
					if err != nil {
						return
					}
				}
			}()

			// Client sends AYT
			_, err := clientConn.Write([]byte{telnet.IAC, telnet.AYT})
			Expect(err).NotTo(HaveOccurred())

			buf := make([]byte, 1024)
			n, err := clientConn.Read(buf)
			Expect(err).NotTo(HaveOccurred())

			// Expect [Yes]
			expected := []byte("\r\n[Yes]\r\n")
			Expect(buf[:n]).To(Equal(expected))
		})
	})

	Context("Sub-negotiation", func() {
		It("should parse NAWS data", func() {
			go func() {
				defer GinkgoRecover()
				buf := make([]byte, 1024)
				for {
					_, err := connection.Read(buf)
					if err != nil {
						return
					}
				}
			}()

			// Negotiate NAWS
			// We simulate that we already agreed to DO NAWS
			connection.EnableRemoteOption(telnet.NAWS)

			// Send Sub-negotiation: IAC SB NAWS 0 80 0 24 IAC SE
			// 80 = 0x50, 24 = 0x18
			data := []byte{
				telnet.IAC, telnet.SB, telnet.NAWS,
				0, 80, 0, 24,
				telnet.IAC, telnet.SE,
			}
			_, err := clientConn.Write(data)
			Expect(err).NotTo(HaveOccurred())

			// Give it a moment to process
			Eventually(func() int {
				return connection.WindowWidth
			}, 1*time.Second).Should(Equal(80))

			Expect(connection.WindowHeight).To(Equal(24))
		})
	})
})
