CREATE TABLE orders (
            order_uid VARCHAR(50) PRIMARY KEY,
            track_number VARCHAR(50),
            entry VARCHAR(10),
            locale VARCHAR(10),
            internal_signature VARCHAR(50),
            customer_id VARCHAR(50),
            delivery_service VARCHAR(50),
            shardkey VARCHAR(10),
            sm_id INTEGER,
            date_created TIMESTAMP,
            oof_shard VARCHAR(10)
        );

        CREATE TABLE delivery (
            order_uid VARCHAR(50) PRIMARY KEY,
            name VARCHAR(100),
            phone VARCHAR(20),
            zip VARCHAR(20),
            city VARCHAR(100),
            address VARCHAR(200),
            region VARCHAR(100),
            email VARCHAR(100),
            FOREIGN KEY (order_uid) REFERENCES orders(order_uid)  
        );

        CREATE TABLE payment (
            order_uid VARCHAR(50) PRIMARY KEY,
            transaction VARCHAR(50),
            request_id VARCHAR(50),
            currency VARCHAR(10),
            provider VARCHAR(50),
            amount INTEGER,
            payment_dt BIGINT,
            bank VARCHAR(50),
            delivery_cost INTEGER,
            goods_total INTEGER,
            custom_fee INTEGER,
            FOREIGN KEY (order_uid) REFERENCES orders(order_uid)
        );

        CREATE TABLE items (
            id SERIAL PRIMARY KEY,
            order_uid VARCHAR(50),
            chrt_id INTEGER,
            track_number VARCHAR(50),
            price INTEGER,
            rid VARCHAR(50),
            name VARCHAR(100),
            sale INTEGER,
            size VARCHAR(10),
            total_price INTEGER,
            nm_id INTEGER,
            brand VARCHAR(100),
            status INTEGER,
            FOREIGN KEY (order_uid) REFERENCES orders(order_uid)
        );