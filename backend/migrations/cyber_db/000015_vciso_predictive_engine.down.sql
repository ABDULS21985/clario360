ALTER TABLE IF EXISTS vciso_feature_snapshots DISABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS vciso_predictions DISABLE ROW LEVEL SECURITY;

DROP TABLE IF EXISTS vciso_feature_snapshots;
DROP TABLE IF EXISTS vciso_prediction_models;
DROP TABLE IF EXISTS vciso_predictions;

