package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
)

var password string

var hashPasswordCmd = &cobra.Command{
	Use:   "hash-password",
	Short: "Generate bcrypt hash for a password",
	Run: func(cmd *cobra.Command, args []string) {
		if password == "" {
			fmt.Println("Error: --password flag is required")
			return
		}

		// Генерируем хеш с ценой 10 (стандарт 2026)
		hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
		if err != nil {
			fmt.Printf("Error generating hash: %v\n", err)
			return
		}

		fmt.Println("\nGenerated Bcrypt Hash for your DB:")
		fmt.Println(string(hash))
	},
}

func init() {
	// Добавляем команду в корень приложения
	RootCmd.AddCommand(hashPasswordCmd)

	// Добавляем флаг для ввода пароля
	hashPasswordCmd.Flags().StringVarP(&password, "password", "p", "", "Password to hash")
}
