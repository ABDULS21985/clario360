-- Migration 000026: Fix vciso_integrations seed data contract mismatches
-- Corrects invalid enum values in type, sync_frequency, and status columns
-- that were inserted by migration 000016 using values outside the validated set.
-- Safe to apply on any DB that has already run 000016.

-- ── Fix type column ────────────────────────────────────────────────────────
-- 'edr' is not a valid CyberIntegrationType; map to cloud_security
UPDATE vciso_integrations SET type = 'cloud_security' WHERE type = 'edr';
-- 'itsm' is not a valid CyberIntegrationType; map to ticketing
UPDATE vciso_integrations SET type = 'ticketing' WHERE type = 'itsm';
-- 'vulnerability_scanner' is not a valid CyberIntegrationType; map to asset_management
UPDATE vciso_integrations SET type = 'asset_management' WHERE type = 'vulnerability_scanner';

-- ── Fix sync_frequency column ─────────────────────────────────────────────
-- 'realtime' is not a valid sync frequency; map to every_5m (closest approximation)
UPDATE vciso_integrations SET sync_frequency = 'every_5m' WHERE sync_frequency = 'realtime';
-- 'every_5min' is not a valid sync frequency; map to every_5m
UPDATE vciso_integrations SET sync_frequency = 'every_5m' WHERE sync_frequency = 'every_5min';
-- 'hourly' is not a valid sync frequency; map to every_hour
UPDATE vciso_integrations SET sync_frequency = 'every_hour' WHERE sync_frequency = 'hourly';

-- ── Fix status column ─────────────────────────────────────────────────────
-- 'active' is not a valid CyberIntegrationStatus; map to connected
UPDATE vciso_integrations SET status = 'connected' WHERE status = 'active';
-- 'inactive' is not a valid CyberIntegrationStatus; map to disconnected
UPDATE vciso_integrations SET status = 'disconnected' WHERE status = 'inactive';
