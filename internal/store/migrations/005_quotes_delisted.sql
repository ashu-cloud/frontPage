ALTER TABLE quotes ADD COLUMN delisted INTEGER NOT NULL DEFAULT 0;

UPDATE quotes SET delisted = 1 WHERE symbol IN ('BBI', 'LEH', 'BSC', 'ENE', 'WCOM', 'CC', 'TOYS', 'SHLD', 'PCLN', 'TWTR', 'ATVI', 'VMW');
