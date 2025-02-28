#!/bin/bash

#  Skrip ini akan membuat file migrasi baru dengan nama yang diberikan sebagai argumen. Ini akan membuat file migrasi baru di direktori internal/migration dan menambahkan template dasar untuk fungsi migrasi.
#  Untuk menggunakan skrip ini, Anda perlu memberikan izin eksekusi ke skrip tersebut:
#  chmod +x script/create_migration.sh
#
#  jalankan script untuk membuat file migrasi baru dengan nama yang diberikan sebagai argumen:
#  ./script/create_migration.sh create_users_table

# Periksa apakah nama migrasi diberikan sebagai argumen
if [ -z "$1" ]; then
  echo "Usage: $0 migration_name"
  exit 1
fi

# Mendapatkan timestamp saat ini dalam format YYYYMMDDHHMMSS
timestamp=$(date +"%Y%m%d%H%M%S")

# Nama migrasi dari argumen pertama
migration_name=$1

# Nama file migrasi dengan format timestamp_migration_name.go
filename="${timestamp}_${migration_name}.go"

# Nama fungsi migrasi: ubah menjadi camelCase
function_name="$(echo "${migration_name}" | awk -F_ '{for (i=1; i<=NF; i++) {if (i == 1) {printf tolower($i)} else {printf toupper(substr($i,1,1)) tolower(substr($i,2))}}}')Migration"

# ID migrasi dengan format timestamp_migration_name
migration_id="${timestamp}_${migration_name}"

# Direktori tempat file migrasi akan dibuat
migration_dir="database/migrations"

# Periksa apakah direktori migrasi ada
if [ ! -d "$migration_dir" ]; then
  echo "Migration directory $migration_dir does not exist. Creating..."
  mkdir -p "$migration_dir"
fi

# Buat file migrasi di direktori yang ditentukan
touch "${migration_dir}/${filename}"

# Tambahkan template dasar ke file migrasi baru
cat <<EOL > "${migration_dir}/${filename}"
package migrations

import (
	"gorm.io/gorm"
	"github.com/go-gormigrate/gormigrate/v2"
)

// Fungsi migrasi untuk $migration_name
func $function_name() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "$migration_id",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate()
		},
		Rollback: func(tx *gorm.DB) error {
		  return tx.Migrator().DropTable("")
		},
	}
}
EOL

echo "Created migration file: ${migration_dir}/${filename}"