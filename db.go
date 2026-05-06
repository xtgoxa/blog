package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

// DB is the global database connection
var DB *sql.DB

// InitDB initializes database connection
func InitDB() error {
	config := AppConfig

	if config.DBPassword == "" {
		return fmt.Errorf("DB_PASSWORD environment variable is required")
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.DBUser,
		config.DBPassword,
		config.DBHost,
		config.DBPort,
		config.DBName,
	)

	var err error
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err = DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)

	// Create tables if not exist
	if err = createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	log.Println("鉁?Database connected successfully!")
	return nil
}

// createTables creates necessary tables
func createTables() error {
	// Users table
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			username VARCHAR(50) UNIQUE NOT NULL,
			email VARCHAR(100) UNIQUE NOT NULL,
			password VARCHAR(255) NOT NULL,
			role VARCHAR(20) DEFAULT 'user',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	if err != nil {
		return err
	}

	// Posts table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS posts (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			title VARCHAR(200) NOT NULL,
			content TEXT NOT NULL,
			author VARCHAR(100),
			category VARCHAR(50),
			tags TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	if err != nil {
		return err
	}

	// Comments table with approval status
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS comments (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			post_id BIGINT NOT NULL,
			user_id BIGINT NOT NULL,
			content TEXT NOT NULL,
			status VARCHAR(20) DEFAULT 'pending',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_post_id (post_id),
			INDEX idx_status (status)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	if err != nil {
		return err
	}

	// Add status column if not exists (for existing tables)
	_, _ = DB.Exec("ALTER TABLE comments ADD COLUMN status VARCHAR(20) DEFAULT 'pending'")
	_, _ = DB.Exec("UPDATE comments SET status = 'approved' WHERE status IS NULL")

	// Create settings table for admin password
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS settings (
			key_name VARCHAR(50) PRIMARY KEY,
			key_value TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	if err != nil {
		return err
	}

	// Insert default admin password if not exists
	var count int
	DB.QueryRow("SELECT COUNT(*) FROM settings WHERE key_name = 'admin_password'").Scan(&count)
	if count == 0 {
		hashedPassword, _ := HashPassword("admin123")
		_, err = DB.Exec("INSERT INTO settings (key_name, key_value) VALUES ('admin_password', ?)", hashedPassword)
		if err != nil {
			return err
		}
		log.Println("鈿狅笍  Default admin password is 'admin123', please change it after first login!")
	}

	// Carousel slides table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS carousel (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			title VARCHAR(200) NOT NULL,
			subtitle TEXT,
			button_text VARCHAR(50),
			button_link VARCHAR(255),
			image_url VARCHAR(500),
			sort_order INT DEFAULT 0,
			is_active BOOLEAN DEFAULT TRUE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	if err != nil {
		return err
	}

	// Insert default carousel slides if empty
	DB.QueryRow("SELECT COUNT(*) FROM carousel").Scan(&count)
	if count == 0 {
		_, err = DB.Exec(`
			INSERT INTO carousel (title, subtitle, button_text, button_link, image_url, sort_order, is_active) VALUES
			('娆㈣繋鏉ュ埌鎴戠殑鍗氬', '鎺㈢储鎶€鏈笘鐣岋紝鍒嗕韩缂栫▼蹇冨緱锛岃褰曟垚闀胯冻杩?, '浜嗚В鏇村', '/about', 'https://picsum.photos/1200/400?random=1', 1, TRUE),
			('鎶€鏈垎浜?, 'Go銆乄eb寮€鍙戙€佹暟鎹簱銆佷簯璁＄畻绛夋妧鏈枃绔?, '鏌ョ湅鏂囩珷', '/?category=Tech', 'https://picsum.photos/1200/400?random=2', 2, TRUE),
			('鍔犲叆鎴戜滑', '娉ㄥ唽璐﹀彿锛屽弬涓庤瘎璁轰簰鍔紝鍒嗕韩浣犵殑鎯虫硶', '绔嬪嵆娉ㄥ唽', '/register', 'https://picsum.photos/1200/400?random=3', 3, TRUE)
		`)
		if err != nil {
			return err
		}
	}

	return nil
}

// CloseDB closes database connection
func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}

// GetDB returns the database connection
func GetDB() *sql.DB {
	return DB
}
