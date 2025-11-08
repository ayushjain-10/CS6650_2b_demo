-- E-commerce Database Schema for Shopping Cart System
-- Optimized for <50ms cart retrieval and 100 concurrent users

CREATE DATABASE IF NOT EXISTS ecommerce;
USE ecommerce;

-- Table 1: Customers
CREATE TABLE IF NOT EXISTS customers (
    customer_id VARCHAR(50) PRIMARY KEY,
    email VARCHAR(255) UNIQUE,
    full_name VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Table 2: Carts
CREATE TABLE IF NOT EXISTS carts (
    cart_id VARCHAR(50) PRIMARY KEY,
    customer_id VARCHAR(50) NOT NULL,
    status ENUM('active', 'checked_out', 'abandoned') DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_customer_status (customer_id, status),
    INDEX idx_updated_at (updated_at),
    
    FOREIGN KEY (customer_id) REFERENCES customers(customer_id) 
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Table 3: Cart Items
CREATE TABLE IF NOT EXISTS cart_items (
    item_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    cart_id VARCHAR(50) NOT NULL,
    product_id VARCHAR(50) NOT NULL,
    product_name VARCHAR(255) NOT NULL,
    quantity INT NOT NULL DEFAULT 1,
    price_per_unit DECIMAL(10, 2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_cart_id (cart_id),
    UNIQUE KEY unique_cart_product (cart_id, product_id),
    
    FOREIGN KEY (cart_id) REFERENCES carts(cart_id) 
        ON DELETE CASCADE,
    
    CONSTRAINT chk_quantity CHECK (quantity > 0),
    CONSTRAINT chk_price CHECK (price_per_unit >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Set recommended transaction isolation level for shopping carts
SET SESSION TRANSACTION ISOLATION LEVEL READ COMMITTED;

