package normalize

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

var latinMinorWords = map[string]bool{
	"a": true, "an": true, "the": true,
	"and": true, "or": true, "but": true, "nor": true,
	"so": true, "yet": true,
	"in": true, "on": true, "at": true, "to": true,
	"for": true, "of": true, "with": true, "by": true,
	"from": true, "as": true, "into": true, "like": true,
	"near": true, "over": true, "upon": true, "via": true,
}

var cyrillicProperNouns = map[string]bool{
	// ====== Имена (существующие) ======
	"александр": true, "алексей": true, "андрей": true, "анна": true,
	"антон": true, "артём": true, "борис": true, "вадим": true,
	"валерий": true, "василий": true, "виктор": true, "виталий": true,
	"владимир": true, "вячеслав": true, "геннадий": true, "георгий": true,
	"григорий": true, "дарья": true, "денис": true, "дмитрий": true,
	"евгений": true, "екатерина": true, "елена": true, "иван": true,
	"игорь": true, "ирина": true, "кирилл": true, "константин": true,
	"лариса": true, "леонид": true, "максим": true, "мария": true,
	"марина": true, "михаил": true, "наталья": true, "никита": true,
	"николай": true, "олег": true, "ольга": true, "павел": true,
	"пётр": true, "роман": true, "руслан": true, "светлана": true,
	"сергей": true, "станислав": true, "степан": true, "татьяна": true,
	"тимур": true, "фёдор": true, "юлия": true, "юрий": true,
	"ярослав": true,

	// ====== Страны и континенты ======
	"азербайджан": true, "албания": true, "англия": true, "аргентина": true,
	"армения": true, "африка": true, "беларусь": true, "бельгия": true,
	"болгария": true, "бразилия": true, "венгрия": true, "вьетнам": true,
	"германия": true, "греция": true, "грузия": true, "дания": true,
	"египет": true, "израиль": true, "индия": true, "индонезия": true,
	"иордания": true, "ирак": true, "иран": true, "ирландия": true,
	"испания": true, "италия": true, "казахстан": true, "канада": true,
	"кипр": true, "китай": true, "колумбия": true, "корея": true,
	"куба": true, "латвия": true, "литва": true, "мексика": true,
	"молдова": true, "монголия": true, "нидерланды": true, "новгород": true,
	"норвегия": true, "польша": true, "португалия": true, "россия": true,
	"румыния": true, "сербия": true, "сирия": true, "словакия": true,
	"словения": true, "ссср": true, "сша": true, "таджикистан": true,
	"таиланд": true, "турция": true, "узбекистан": true, "украина": true,
	"финляндия": true, "франция": true, "хорватия": true, "черногория": true,
	"чехия": true, "чили": true, "швейцария": true, "швеция": true,
	"эстония": true, "югославия": true, "япония": true,

	// ====== Города России и СНГ (крупные + культурные) ======
	"абакан": true, "анапа": true, "архангельск": true, "астрахань": true,
	"баку": true, "барнаул": true, "белгород": true, "бишкек": true,
	"брянск": true, "великийновгород": true, "владивосток": true,
	"владикавказ": true, "волгоград": true,
	"вологда": true, "воронеж": true, "выборг": true, "вязьма": true,
	"грозный": true, "дербент": true, "днепр": true, "донецк": true,
	"душанбе": true, "екатеринбург": true, "ереван": true,
	"ессентуки": true, "железноводск": true, "житомир": true,
	"запорожье": true, "зеленоград": true, "иваново": true,
	"ижевск": true, "иркутск": true, "казань": true, "калининград": true,
	"калуга": true, "каменецподольский": true, "каменск": true,
	"караганда": true, "кемерово": true, "киев": true, "киров": true,
	"кишинёв": true, "кострома": true, "краснодар": true,
	"красноярск": true, "кременчуг": true, "кривойрог": true,
	"курган": true, "курск": true, "кызыл": true, "львов": true,
	"магадан": true, "магнитогорск": true, "майкоп": true,
	"махачкала": true, "минск": true, "москва": true,
	"мурманск": true, "мытищи": true, "набережныечелны": true,
	"назрань": true, "нальчик": true, "нижневартовск": true,
	"нижнийновгород": true, "николаев": true, "новокузнецк": true,
	"новороссийск": true, "новосибирск": true, "новочеркасск": true,
	"норильск": true, "ноябрьск": true, "обнинск": true, "одесса": true,
	"омск": true, "орёл": true, "оренбург": true, "павлодар": true,
	"пенза": true, "пермь": true, "петрозаводск": true, "питер": true,
	"полтава": true, "псков": true, "пятигорск": true, "рига": true,
	"ростов": true, "рубцовск": true, "рязань": true, "самара": true,
	"санктпетербург": true, "саранск": true, "саратов": true,
	"севастополь": true, "сергиевпосад": true, "симферополь": true,
	"смоленск": true, "сочи": true, "ставрополь": true,
	"старыйос-кол": true, "суздаль": true, "сургут": true,
	"сыктывкар": true, "тамбов": true, "тверь": true,
	"тбилиси": true, "тернополь": true, "тобольск": true,
	"томск": true, "тула": true, "тюмень": true, "улан-удэ": true,
	"ульяновск": true, "уфа": true, "хабаровск": true, "харьков": true,
	"хибины": true, "хмельницкий": true, "черкассы": true,
	"череповец": true, "чернигов": true, "челябинск": true,
	"чита": true, "элиста": true, "южносахалинск": true,
	"якутск": true, "ялта": true, "ярославль": true,

	// ====== Города мира ======
	"абудаби": true, "амстердам": true, "анкара": true, "афины": true,
	"багдад": true, "бангкок": true, "барселона": true, "батуми": true,
	"берлин": true, "берн": true, "брюссель": true, "будапешт": true,
	"буэнос-айрес": true, "вашингтон": true, "вена": true,
	"венция": true, "вильнюс": true, "гавана": true, "гаага": true,
	"дамаск": true, "дубай": true, "дублин": true, "женевa": true,
	"иерусалим": true, "каир": true, "кейптаун": true, "копенгаген": true,
	"лас-вегас": true, "ливерпуль": true, "лисабон": true,
	"лондон": true, "лос-анджелес": true, "люксембург": true,
	"мадрид": true, "милан": true, "монтевидео": true, "мюнхен": true,
	"осло": true, "париж": true, "прага": true, "рейкьявик": true,
	"рим": true, "салоники": true, "сан-франциско": true,
	"сент-луис": true, "стокгольм": true, "сydney": true, "тель-авив": true,
	"токио": true, "триполи": true, "ханой": true, "хельсинки": true,
	"чикаго": true, "шанхай": true,

	// ====== Реки, моря, озера ======
	"амур": true, "ангара": true, "байкал": true, "балтика": true,
	"волга": true, "днестр": true, "дон": true,
	"дунай": true, "енисей": true, "ильмень": true, "иртыш": true,
	"кама": true, "каспий": true, "ладога": true, "лена": true,
	"нева": true, "обь": true, "онега": true, "печора": true,
	"севернаядвина": true, "средиземное": true, "сухона": true,
	"урал": true, "черное": true, "чукотка": true, "яуза": true,

	// ====== Острова, регионы, горы ======
	"алтай": true, "кавказ": true, "камчатка": true,
	"карелия": true, "крым": true, "курилы": true, "сахалин": true,
	"сибирь": true, "таймыр": true, "ямал": true,

	// ====== Мифология, литература, кино ======
	"аполлон": true, "артемон": true, "ахилл": true, "буратино": true,
	"гамлет": true, "геркулес": true, "донкихот": true,
	"донжуан": true, "зеус": true, "иисус": true, "каин": true,
	"кашпировский": true, "квазимодо": true, "королева": true,
	"леший": true, "люцифер": true, "маугли": true, "мессия": true,
	"нарцисс": true, "одиссей": true, "олимп": true, "орфей": true,
	"прометей": true, "робинзон": true, "робингуд": true,
	"сатана": true, "снегурочка": true, "спартак": true,
	"тарасбульба": true, "фауст": true, "цербер": true,
	"чебурашка": true, "шехеразада": true, "шива": true, "эдип": true,

	// ====== Бренды / имена нарицательные как собственные (в песнях) ======
	"кока-кола": true, "мерседес": true, "мицубиси": true, "ниссан": true,
	"пепси": true, "роллс-ройс": true, "тойота": true, "фольксваген": true,
	"форд": true, "хаммер": true, "шкода": true,
}

var cyrillicAbbreviations = map[string]string{
	"ддт": "ДДТ", "би-2": "Би-2", "аукцыон": "АукцЫон",
}

func collapseSpaces(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prev := ' '
	for _, r := range s {
		if unicode.IsSpace(r) {
			if prev != ' ' {
				b.WriteRune(' ')
			}
			prev = ' '
		} else {
			b.WriteRune(r)
			prev = r
		}
	}
	return strings.TrimSpace(b.String())
}

func isCyrillic(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Cyrillic, r) {
			return true
		}
	}
	return false
}

func capitalizeCyrillicWord(word string, isFirst bool) string {
	lower := strings.ToLower(word)

	if repl, ok := cyrillicAbbreviations[lower]; ok {
		return repl
	}

	if cyrillicProperNouns[lower] || isFirst {
		r, size := utf8.DecodeRuneInString(lower)
		if r == utf8.RuneError {
			return lower
		}
		return string(unicode.ToUpper(r)) + lower[size:]
	}

	return lower
}

func capitalizeLatinWord(word string, isFirst bool) string {
	lower := strings.ToLower(word)
	if !isFirst && latinMinorWords[lower] {
		return lower
	}
	r, size := utf8.DecodeRuneInString(lower)
	if r == utf8.RuneError {
		return lower
	}
	return string(unicode.ToUpper(r)) + lower[size:]
}

func SongName(raw string) string {
	s := collapseSpaces(raw)
	if s == "" {
		return s
	}

	words := strings.Split(s, " ")
	cyrillic := isCyrillic(s)

	for i, w := range words {
		if cyrillic {
			words[i] = capitalizeCyrillicWord(w, i == 0)
		} else {
			words[i] = capitalizeLatinWord(w, i == 0)
		}
	}

	return strings.Join(words, " ")
}
