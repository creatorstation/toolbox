package models

// InfluencerPost represents the n8n_influencer_posts table
type InfluencerPost struct {
	ID            string   `gorm:"primaryKey"`
	AccountID     uint   `gorm:"column:account_id"`
	VideoURL      string `gorm:"column:video_url"`
	Transcription string `gorm:"column:transcription"`
	TakenAt       string `gorm:"column:taken_at"`
}