-- Reverse of 000008_catalog_sections.up.sql
-- Drop in reverse FK order: section_teachers depends on sections.

DROP TABLE IF EXISTS section_teachers;
DROP TABLE IF EXISTS sections;
