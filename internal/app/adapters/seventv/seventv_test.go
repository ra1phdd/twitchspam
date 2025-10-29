package seventv_test

//func TestEmoteStats(t *testing.T) {
//	t.Parallel()
//
//	manager, err := config.New()
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	a := api.NewTwitch(logger.New(), manager, &http.Client{}, 5)
//	channelID, err := a.GetChannelID("stintik")
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	cacheStats := storage.NewCache[stream.SessionStats](0, 0, false, false, "", 0)
//	s := stream.NewStream("stintik", file_server.New(logger.New(), &http.Client{}), cacheStats)
//	s.SetChannelID(channelID)
//
//	sv := seventv.New(logger.New(), manager.Get(), s, &http.Client{})
//
//	tests := []struct {
//		name        string
//		words       []string
//		wantCount   int
//		wantOnlyEmo bool
//	}{
//		{
//			name:        "Empty slice",
//			words:       []string{},
//			wantCount:   0,
//			wantOnlyEmo: false,
//		},
//		{
//			name:        "0",
//			words:       []string{"0", "0", "0"},
//			wantCount:   3,
//			wantOnlyEmo: true,
//		},
//		{
//			name:        "Only emotes above threshold",
//			words:       []string{")", "0", "...."},
//			wantCount:   3,
//			wantOnlyEmo: true,
//		},
//		{
//			name:        "Mixed emotes and text below threshold",
//			words:       []string{"0", "hiiiiii", ")"},
//			wantCount:   2,
//			wantOnlyEmo: false,
//		},
//		{
//			name:        "Text only",
//			words:       []string{"hiiiiiiiii", "world"},
//			wantCount:   0,
//			wantOnlyEmo: false,
//		},
//		{
//			name:        "With extra spaces",
//			words:       []string{"  0  ", "   ", "test "},
//			wantCount:   1,
//			wantOnlyEmo: false,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			t.Parallel()
//
//			count, onlyEmotes := sv.EmoteStats(tt.words)
//			if count != tt.wantCount {
//				t.Errorf("count = %d, want %d", count, tt.wantCount)
//			}
//			if onlyEmotes != tt.wantOnlyEmo {
//				t.Errorf("onlyEmotes = %v, want %v", onlyEmotes, tt.wantOnlyEmo)
//			}
//		})
//	}
//}
