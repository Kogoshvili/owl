package intelligence

import (
	"database/sql"
	"log/slog"
	"math"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

type Keyword struct {
	Term  string
	Score float64
}

type Corpus struct {
	Keywords   map[int64][]Keyword
	KeywordMap map[int64]map[string]float64
}

var stopwords = map[string]bool{
	"the": true, "and": true, "or": true, "but": true, "if": true,
	"then": true, "else": true, "of": true, "at": true,
	"by": true, "for": true, "with": true, "about": true, "is": true,
	"am": true, "are": true, "was": true, "were": true, "be": true,
	"been": true, "being": true, "have": true, "has": true, "had": true,
	"do": true, "does": true, "did": true, "can": true, "could": true,
	"will": true, "would": true, "shall": true, "should": true,
	"may": true, "might": true, "must": true, "this": true, "that": true,
	"these": true, "those": true, "a": true, "an": true, "as": true,
	"in": true, "on": true, "to": true, "from": true, "into": true,
	"through": true, "i": true, "you": true, "he": true, "she": true,
	"it": true, "we": true, "they": true, "my": true, "your": true,
	"his": true, "her": true, "its": true, "our": true, "their": true,
	"what": true, "which": true, "who": true, "whom": true, "whose": true,
	"where": true, "why": true, "how": true,
	// Numbers as strings
	"0": true, "1": true, "2": true, "3": true, "4": true,
	"5": true, "6": true, "7": true, "8": true, "9": true,
	"10": true, "11": true, "12": true, "13": true, "14": true,
	"15": true, "16": true, "17": true, "18": true, "19": true,
	"20": true, "30": true, "40": true, "50": true,
	"100": true, "1000": true,
}

var wordRegex = regexp.MustCompile(`[a-zA-Z0-9]+`)

func Tokenize(text string) []string {
	matches := wordRegex.FindAllString(text, -1)
	return matches
}

func NormalizeTerm(term string) string {
	return strings.ToLower(strings.TrimSpace(term))
}

func IsStopword(term string) bool {
	return stopwords[term]
}

func IsNumeric(term string) bool {
	for _, r := range term {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return len(term) > 0
}

func ExtractKeywordsFromContent(content string, topN int) []Keyword {
	tokens := Tokenize(content)
	termFreq := make(map[string]int)

	for _, token := range tokens {
		term := NormalizeTerm(token)
		if len(term) < 3 {
			continue
		}
		if IsStopword(term) {
			continue
		}
		if IsNumeric(term) {
			continue
		}
		termFreq[term]++
	}

	keywords := make([]Keyword, 0, len(termFreq))
	for term, freq := range termFreq {
		keywords = append(keywords, Keyword{
			Term:  term,
			Score: float64(freq),
		})
	}

	return topKeywords(keywords, topN)
}

func topKeywords(keywords []Keyword, n int) []Keyword {
	if n <= 0 {
		return keywords
	}
	if len(keywords) <= n {
		return keywords
	}

	for i := 0; i < n; i++ {
		maxIdx := i
		for j := i + 1; j < len(keywords); j++ {
			if keywords[j].Score > keywords[maxIdx].Score {
				maxIdx = j
			}
		}
		keywords[i], keywords[maxIdx] = keywords[maxIdx], keywords[i]
	}

	return keywords[:n]
}

type Analyzer struct {
	db *sql.DB
}

func NewAnalyzer(db *sql.DB) *Analyzer {
	return &Analyzer{db: db}
}

func (a *Analyzer) GetFileKeywords(fileID int64, topN int) ([]Keyword, error) {
	var content sql.NullString
	err := a.db.QueryRow(`SELECT content FROM files_fts WHERE file_id = ?`, fileID).Scan(&content)
	if err != nil {
		if err == sql.ErrNoRows {
			return []Keyword{}, nil
		}
		return nil, err
	}

	if !content.Valid || content.String == "" {
		return []Keyword{}, nil
	}

	return ExtractKeywordsFromContent(content.String, topN), nil
}

func (a *Analyzer) GetFileKeywordsWithFallback(fileID int64, topN int) ([]Keyword, error) {
	var keywords []Keyword

	// Try content first
	var content sql.NullString
	err := a.db.QueryRow(`SELECT content FROM files_fts WHERE file_id = ?`, fileID).Scan(&content)
	if err == nil && content.Valid && content.String != "" {
		keywords = ExtractKeywordsFromContent(content.String, topN)
	}

	// Always add filename keywords + extension tag (even when content exists)
	var name string
	err = a.db.QueryRow(`SELECT name FROM files WHERE id = ?`, fileID).Scan(&name)
	if err == nil {
		baseName := strings.TrimSuffix(name, filepath.Ext(name))
		for _, token := range Tokenize(baseName) {
			term := NormalizeTerm(token)
			if len(term) < 3 || IsStopword(term) || IsNumeric(term) {
				continue
			}
			keywords = append(keywords, Keyword{Term: term, Score: 1.0})
		}
	}

	// Add extension tag with higher weight
	var ext string
	a.db.QueryRow(`SELECT extension FROM files WHERE id = ?`, fileID).Scan(&ext)
	if tag := getExtensionTag(ext); tag != "" {
		keywords = append(keywords, Keyword{Term: tag, Score: 5.0})
	}

	return topKeywords(keywords, topN), nil
}

func (a *Analyzer) BuildCorpus(fileIDs []int64) (*Corpus, error) {
	corpusKeywords, err := a.BuildCorpusTFIDFWithFallback(fileIDs)
	if err != nil {
		return nil, err
	}

	keywordMap := make(map[int64]map[string]float64)
	for _, id := range fileIDs {
		kwMap := make(map[string]float64)
		for _, kw := range corpusKeywords[id] {
			kwMap[kw.Term] = kw.Score
		}
		keywordMap[id] = kwMap
	}

	return &Corpus{
		Keywords:   corpusKeywords,
		KeywordMap: keywordMap,
	}, nil
}

func (a *Analyzer) BuildCorpusTFIDF(fileIDs []int64) (map[int64][]Keyword, error) {
	result := make(map[int64][]Keyword)
	termDocFreq := make(map[string]int)
	fileKeywordsMap := make(map[int64]map[string]int)

	slog.Info("build-corpus: fetching keywords", "files", len(fileIDs))
	for i, fileID := range fileIDs {
		keywords, err := a.GetFileKeywords(fileID, 100)
		if err != nil {
			return nil, err
		}

		keywordMap := make(map[string]int)
		for _, kw := range keywords {
			keywordMap[kw.Term] = int(kw.Score)
			termDocFreq[kw.Term]++
		}

		fileKeywordsMap[fileID] = keywordMap

		if (i+1)%50 == 0 || i == len(fileIDs)-1 {
			slog.Info("build-corpus: progress", "files", i+1, "total", len(fileIDs), "unique_terms_so_far", len(termDocFreq))
		}
	}

	totalDocs := len(fileIDs)

	for _, fileID := range fileIDs {
		keywordMap := fileKeywordsMap[fileID]
		keywords := make([]Keyword, 0, len(keywordMap))

		for term, freq := range keywordMap {
			docFreq := termDocFreq[term]
			idf := math.Log(float64(totalDocs) / float64(docFreq))
			tfidf := float64(freq) * idf

			keywords = append(keywords, Keyword{
				Term:  term,
				Score: tfidf,
			})
		}

		result[fileID] = topKeywords(keywords, 20)
	}

	slog.Info("build-corpus: complete", "files", totalDocs, "unique_terms", len(termDocFreq))

	return result, nil
}

func (a *Analyzer) BuildCorpusTFIDFWithFallback(fileIDs []int64) (map[int64][]Keyword, error) {
	result := make(map[int64][]Keyword)
	termDocFreq := make(map[string]int)
	fileKeywordsMap := make(map[int64]map[string]int)

	slog.Info("build-corpus-with-fallback: starting", "files", len(fileIDs))
	for i, fileID := range fileIDs {
		keywords, err := a.GetFileKeywordsWithFallback(fileID, 100)
		if err != nil {
			return nil, err
		}

		keywordMap := make(map[string]int)
		for _, kw := range keywords {
			keywordMap[kw.Term] = int(kw.Score)
			termDocFreq[kw.Term]++
		}

		fileKeywordsMap[fileID] = keywordMap

		if (i+1)%100 == 0 || i == len(fileIDs)-1 {
			slog.Info("build-corpus-with-fallback: progress", "files", i+1, "total", len(fileIDs), "unique_terms_so_far", len(termDocFreq))
		}
	}

	totalDocs := len(fileIDs)

	for _, fileID := range fileIDs {
		keywordMap := fileKeywordsMap[fileID]
		keywords := make([]Keyword, 0, len(keywordMap))

		for term, freq := range keywordMap {
			docFreq := termDocFreq[term]
			idf := math.Log(float64(totalDocs) / float64(docFreq))
			tfidf := float64(freq) * idf

			keywords = append(keywords, Keyword{
				Term:  term,
				Score: tfidf,
			})
		}

		result[fileID] = topKeywords(keywords, 20)
	}

	slog.Info("build-corpus: complete", "files", totalDocs, "unique_terms", len(termDocFreq))

	return result, nil
}

func (a *Analyzer) GetFileNames(fileIDs []int64) (map[int64]string, error) {
	if len(fileIDs) == 0 {
		return make(map[int64]string), nil
	}

	placeholders := make([]string, len(fileIDs))
	args := make([]any, len(fileIDs))
	for i, id := range fileIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := `SELECT id, name FROM files WHERE id IN (` + strings.Join(placeholders, ",") + `)`
	rows, err := a.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64]string)
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		result[id] = name
	}

	return result, rows.Err()
}