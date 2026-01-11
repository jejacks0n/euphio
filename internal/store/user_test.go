package store_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"euphio/internal/store"
)

var _ = Describe("User Model", func() {
	var db *store.Store

	BeforeEach(func() {
		var err error
		db, err = store.New(":memory:", true)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("CreateUser", func() {
		Context("with valid input", func() {
			It("creates a user successfully", func() {
				err := db.CreateUser("testuser", "password123")
				Expect(err).NotTo(HaveOccurred())

				user, err := db.FindUserByUsername("testuser")
				Expect(err).NotTo(HaveOccurred())
				Expect(user).NotTo(BeNil())
			})
		})

		Context("with a duplicate username", func() {
			It("returns an error", func() {
				_ = db.CreateUser("dupe", "pass")
				err := db.CreateUser("dupe", "pass")
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Authenticate", func() {
		BeforeEach(func() {
			_ = db.CreateUser("validuser", "secretpass")
		})

		It("authenticates with correct credentials", func() {
			user, err := db.Authenticate("validuser", "secretpass")
			Expect(err).NotTo(HaveOccurred())
			Expect(user.Username).To(Equal("validuser"))
		})

		It("fails with incorrect password", func() {
			_, err := db.Authenticate("validuser", "wrongpass")
			Expect(err).To(MatchError("invalid password"))
		})

		It("fails with unknown username", func() {
			_, err := db.Authenticate("ghostinthemachine", "pass")
			Expect(err).To(MatchError("user not found"))
		})
	})
})
