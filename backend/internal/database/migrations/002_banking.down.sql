-- Migration: Remove banking tables
-- 002_banking.down.sql

DROP TABLE IF EXISTS transfers;
DROP TABLE IF EXISTS accounts;
