package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

var (
	ErrRedeemCodeNotFound  = infraerrors.NotFound("REDEEM_CODE_NOT_FOUND", "redeem code not found")
	ErrRedeemCodeUsed      = infraerrors.Conflict("REDEEM_CODE_USED", "redeem code already used")
	ErrInsufficientBalance = infraerrors.BadRequest("INSUFFICIENT_BALANCE", "insufficient balance")
	ErrRedeemRateLimited   = infraerrors.TooManyRequests("REDEEM_RATE_LIMITED", "too many failed attempts, please try again later")
	ErrRedeemCodeLocked    = infraerrors.Conflict("REDEEM_CODE_LOCKED", "redeem code is being processed, please try again")
)

const (
	redeemMaxErrorsPerHour  = 20
	redeemRateLimitDuration = time.Hour
	redeemLockDuration      = 10 * time.Second // 闂備礁銇樺ù鍥╃矓閹绢喖绫嶉柤绋跨仛椤ρ囨⒒閸屾繃褰х紒杈ㄧ箞濮婂ジ宕ｆ径妯哄濠殿喗绻傞…鐑藉绩?
)

// ContextSkipRedeemAffiliate is retained as a no-op compatibility wrapper after
// the legacy redeem-triggered invite rebate flow was deprecated.
func ContextSkipRedeemAffiliate(ctx context.Context) context.Context {
	return ctx
}

// RedeemCache defines cache operations for redeem service
type RedeemCache interface {
	GetRedeemAttemptCount(ctx context.Context, userID int64) (int, error)
	IncrementRedeemAttemptCount(ctx context.Context, userID int64) error

	AcquireRedeemLock(ctx context.Context, code string, ttl time.Duration) (bool, error)
	ReleaseRedeemLock(ctx context.Context, code string) error
}

type RedeemCodeRepository interface {
	Create(ctx context.Context, code *RedeemCode) error
	CreateBatch(ctx context.Context, codes []RedeemCode) error
	GetByID(ctx context.Context, id int64) (*RedeemCode, error)
	GetByCode(ctx context.Context, code string) (*RedeemCode, error)
	Update(ctx context.Context, code *RedeemCode) error
	Delete(ctx context.Context, id int64) error
	Use(ctx context.Context, id, userID int64) error

	List(ctx context.Context, params pagination.PaginationParams) ([]RedeemCode, *pagination.PaginationResult, error)
	ListWithFilters(ctx context.Context, params pagination.PaginationParams, codeType, status, search string) ([]RedeemCode, *pagination.PaginationResult, error)
	ListByUser(ctx context.Context, userID int64, limit int) ([]RedeemCode, error)
	// ListByUserPaginated returns paginated balance/concurrency history for a specific user.
	// codeType filter is optional - pass empty string to return all types.
	ListByUserPaginated(ctx context.Context, userID int64, params pagination.PaginationParams, codeType string) ([]RedeemCode, *pagination.PaginationResult, error)
	// SumPositiveBalanceByUser returns the total recharged amount (sum of positive balance values) for a user.
	SumPositiveBalanceByUser(ctx context.Context, userID int64) (float64, error)
}

// GenerateCodesRequest 闂佹眹鍨婚崰鎰板垂濮樿泛绀傞柟瀵稿仜鎼村﹪鏌ｉ缁樼グ妞ゆ洦鍓欐晥?
type GenerateCodesRequest struct {
	Count int     `json:"count"`
	Value float64 `json:"value"`
	Type  string  `json:"type"`
}

// RedeemCodeResponse 闂佺绻戦崹璺虹暦閺屻儲鍎樺ù锝呮啞閹瑩骞?
type RedeemCodeResponse struct {
	Code      string    `json:"code"`
	Value     float64   `json:"value"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// RedeemService 闂佺绻戦崹璺虹暦閺屻儲鍎樺ù锝堫潐缁犳盯鏌?
type RedeemService struct {
	redeemRepo           RedeemCodeRepository
	userRepo             UserRepository
	subscriptionService  *SubscriptionService
	cache                RedeemCache
	billingCacheService  *BillingCacheService
	entClient            *dbent.Client
	authCacheInvalidator APIKeyAuthCacheInvalidator
	affiliateService     *AffiliateService
}

// NewRedeemService 闂佸憡甯楃粙鎴犵磽閹捐绀傞柟瀵稿仜鎼村﹪鏌ｉ鑽ゅ妽婵犫偓閸ヮ剙绀夐柍銉ョ－閺夎棄銆?
func NewRedeemService(
	redeemRepo RedeemCodeRepository,
	userRepo UserRepository,
	subscriptionService *SubscriptionService,
	cache RedeemCache,
	billingCacheService *BillingCacheService,
	entClient *dbent.Client,
	authCacheInvalidator APIKeyAuthCacheInvalidator,
	affiliateService *AffiliateService,
) *RedeemService {
	return &RedeemService{
		redeemRepo:           redeemRepo,
		userRepo:             userRepo,
		subscriptionService:  subscriptionService,
		cache:                cache,
		billingCacheService:  billingCacheService,
		entClient:            entClient,
		authCacheInvalidator: authCacheInvalidator,
		affiliateService:     affiliateService,
	}
}

// GenerateRandomCode 闂佹眹鍨婚崰鎰板垂濮樿埖鈷曢煫鍥ㄦ⒐缁ㄦ岸鏌涜箛鏂跨仸鐎规洘鐓￠幆?
func (s *RedeemService) GenerateRandomCode() (string, error) {
	// 闂佹眹鍨婚崰鎰板垂?6闁诲孩绋掗〃澶嬩繆椤撱垺鈷曢煫鍥ㄦ⒐缁ㄦ岸鏌℃担鍝勵暭鐎?
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}

	// 闁哄鍎愰崜姘暦閸欏鈻旈柛婵嗗绾锯晠鏌涜箛锝呭绩缂佺粯锕㈠畷姘辨崉閾忚鎲荤紓浣诡殣缂傛氨鎲?
	code := hex.EncodeToString(bytes)

	// 闂佸搫绉堕崢褏妲愰敓鐘茬闁哄稄绠掔粈?XXXX-XXXX-XXXX-XXXX 闂佸搫绉堕崢褏妲?
	parts := []string{
		strings.ToUpper(code[0:8]),
		strings.ToUpper(code[8:16]),
		strings.ToUpper(code[16:24]),
		strings.ToUpper(code[24:32]),
	}

	return strings.Join(parts, "-"), nil
}

// GenerateCodes 闂佸綊娼х紞濠囧闯濞差亝鍋ㄩ柣鏃傤焾閻忓洭鏌涜箛鏂跨仸鐎规洘鐓￠幆?
func (s *RedeemService) GenerateCodes(ctx context.Context, req GenerateCodesRequest) ([]RedeemCode, error) {
	if req.Count <= 0 {
		return nil, errors.New("count must be greater than 0")
	}

	// 闂備緡鍘搁崑鎾绘偣閸ヮ剦妫戦柣鏍ㄧ矌閻氬墽鎷犻懠顑藉亾闁垮鈻旂€广儱顦版禒姗€鎮烽弴姘卞妽闁哄棛鍠栧畷鎰兜妞嬪海顦梺绋跨箲濠€鍦垝椤掑倻灏甸悹鍥皺閳ь剛鍏樺Λ渚€鍩€椤掑倹鍟哄〒姘ｅ亾婵炵⒈浜璺ㄦ崉鐞涒€充壕缁绢參顥撶粈鍕煛閳ь剟顢涘☉妯兼Х闁荤姵鍔楅崰鎰板汲閻斿吋鍋ㄩ柕濞垮€楅懝楣冩⒑椤愮喎浜惧┑鐐叉閹锋繄妲?
	if req.Type != RedeemTypeInvitation && req.Value == 0 {
		return nil, errors.New("value must not be zero")
	}

	if req.Count > 1000 {
		return nil, errors.New("cannot generate more than 1000 codes at once")
	}

	codeType := req.Type
	if codeType == "" {
		codeType = RedeemTypeBalance
	}

	// 闂備緡鍘搁崑鎾绘偣閸ヮ剦妫戦柣鏍ㄧ矌閻氬墽鎷犻懠顑藉亾閻戣姤鍎?value 闁荤姳绀佽ぐ鐐垫嫻?0
	value := req.Value
	if codeType == RedeemTypeInvitation {
		value = 0
	}

	codes := make([]RedeemCode, 0, req.Count)
	for i := 0; i < req.Count; i++ {
		code, err := s.GenerateRandomCode()
		if err != nil {
			return nil, fmt.Errorf("generate code: %w", err)
		}

		codes = append(codes, RedeemCode{
			Code:   code,
			Type:   codeType,
			Value:  value,
			Status: StatusUnused,
		})
	}

	// 闂佸綊娼х紞濠囧闯濞差亜绠甸柟鐑樺灥瀵?
	if err := s.redeemRepo.CreateBatch(ctx, codes); err != nil {
		return nil, fmt.Errorf("create batch codes: %w", err)
	}

	return codes, nil
}

// CreateCode creates a redeem code with caller-provided code value.
// It is primarily used by admin integrations that require an external order ID
// to be mapped to a deterministic redeem code.
func (s *RedeemService) CreateCode(ctx context.Context, code *RedeemCode) error {
	if code == nil {
		return errors.New("redeem code is required")
	}
	code.Code = strings.TrimSpace(code.Code)
	if code.Code == "" {
		return errors.New("code is required")
	}
	if code.Type == "" {
		code.Type = RedeemTypeBalance
	}
	if code.Type != RedeemTypeInvitation && code.Value == 0 {
		return errors.New("value must not be zero")
	}
	if code.Status == "" {
		code.Status = StatusUnused
	}

	if err := s.redeemRepo.Create(ctx, code); err != nil {
		return fmt.Errorf("create redeem code: %w", err)
	}
	return nil
}

// checkRedeemRateLimit 濠碘槅鍋€閸嬫捇鏌＄仦璇插姢闁轰降鍊濋獮瀣暋閺夊灝绠戦梺鐟扮畭閸ㄤ粙寮繝鍕珰妞ゆ牗渚楅崑褔鏌℃担鍝勵暭婵″弶鎮傚畷銉╂晝娴ｇ洅銏ゆ⒒?
func (s *RedeemService) checkRedeemRateLimit(ctx context.Context, userID int64) error {
	if s.cache == nil {
		return nil
	}

	count, err := s.cache.GetRedeemAttemptCount(ctx, userID)
	if err != nil {
		// Redis 闂佸憡鍨跺浠嬪极婵犲洤绫嶉柡鍫㈡暩閻熸繈姊婚崘顏嗗笡妞ゆ帗鐩幃浠嬪Ω閿曗偓閻撴洟鏌熼崹顔拘＄紓?
		return nil
	}

	if count >= redeemMaxErrorsPerHour {
		return ErrRedeemRateLimited
	}

	return nil
}

// incrementRedeemErrorCount 婵犫拃鍛粶濠殿喚鍋ら幃浠嬪Ω閿曗偓閻撴洟鏌涜箛鏂跨仸鐎规洘鐓￠弻銊モ枎閹烘繂娈╅柣鐘辫閸撴繈寮?
func (s *RedeemService) incrementRedeemErrorCount(ctx context.Context, userID int64) {
	if s.cache == nil {
		return
	}

	_ = s.cache.IncrementRedeemAttemptCount(ctx, userID)
}

// acquireRedeemLock 闁诲繐绻戠换鍡涙儊椤栫偞鍤旂€瑰嫭婢樼徊鍧楁煕韫囨柨鐏︾€规洘鐓￠幆宥嗘媴閻戞鏆犻梺鍛婂笒濡绮╁畡閭﹀殨闊洦绋掗弫?
// 闁哄鏅滈弻銊ッ?true 闁荤偞绋忛崝搴ㄥΦ濮樿埖鍤旂€瑰嫭婢樼徊鍧楁煙鐎涙ê濮囧┑顔界洴閺佸秶鈧婧俵se 闁荤偞绋忛崝搴ㄥΦ濮樿埖鐓ュù锝囶焾閸ゆ帡鎮跺鐓庝簻鐎规洑鍗抽幃?
func (s *RedeemService) acquireRedeemLock(ctx context.Context, code string) bool {
	if s.cache == nil {
		return true
	}

	ok, err := s.cache.AcquireRedeemLock(ctx, code, redeemLockDuration)
	if err != nil {
		// Redis 闂佸憡鍨跺浠嬪极婵犲洤绫嶉柡鍫㈡暩閻熸繈姊婚崘顏嗗笡妞ゆ帗鐩獮娆忣吋閸曨厾鈻曢梺鎸庣☉婵傛梻娆㈡搴㈠皫闁哄秲鍔嶅▓鍫曟煙鐠団€虫灈缂併劏浜禒锕傚磼閻愬瓨銆冮梺姹囧妼鐎氼喗鎱ㄩ幖浣哥畱濞达綀顕栧楣冩煛?
		return true
	}
	return ok
}

// releaseRedeemLock 闂備焦褰冮敃銉╁棘娓氣偓瀹曟骞嬮悙鎻掔哎闂佹椿鍠曢懗璺衡枔閹达箑绀嗛柛鈩冾殘椤忓鈧鍠栫换姗€寮?
func (s *RedeemService) releaseRedeemLock(ctx context.Context, code string) {
	if s.cache == nil {
		return
	}

	_ = s.cache.ReleaseRedeemLock(ctx, code)
}

// Redeem 婵炶揪缍€濞夋洟寮妶澶婄闁瑰鍋涙惔濠囨煟?
func (s *RedeemService) Redeem(ctx context.Context, userID int64, code string) (*RedeemCode, error) {
	// 濠碘槅鍋€閸嬫捇鏌＄仦璇插姦婵℃儳鎼湁?
	if err := s.checkRedeemRateLimit(ctx, userID); err != nil {
		return nil, err
	}

	// 闂佸吋鍎抽崲鑼躲亹閸ヮ剙绀嗛柛鈩冾殘椤忓鈧鍠栫换姗€寮ㄩ敐澶嬫櫖鐎光偓閸曨儷鈺傛叏濠靛嫬鐏╅柟顔芥尰缁嬪鍩€椤掑嫬绀傞柟瀵稿仜鎼村﹪鏌ｉ鑽ゎ槮闁艰崵鍠栧畷锝夊箣濠婂啠鏋忛梺?
	if !s.acquireRedeemLock(ctx, code) {
		return nil, ErrRedeemCodeLocked
	}
	defer s.releaseRedeemLock(ctx, code)

	// 闂佸搫琚崕鍙夌珶濮椻偓瀹曟骞嬮悙鎻掔哎闂?
	redeemCode, err := s.redeemRepo.GetByCode(ctx, code)
	if err != nil {
		if errors.Is(err, ErrRedeemCodeNotFound) {
			s.incrementRedeemErrorCount(ctx, userID)
			return nil, ErrRedeemCodeNotFound
		}
		return nil, fmt.Errorf("get redeem code: %w", err)
	}

	// 濠碘槅鍋€閸嬫捇鏌＄仦璇插姎闁告﹩鍓熼獮鎴﹀閵忋垹鐏ｉ梺缁橆焾閸╂牠鍩€?
	if !redeemCode.CanUse() {
		s.incrementRedeemErrorCount(ctx, userID)
		return nil, ErrRedeemCodeUsed
	}

	// 婵°倗濮撮惌渚€鎯佹径鎰闁瑰鍋涙惔濠囨煟椤旀槒鍏岄悶姘煎亰瀹曞湱鈧綆鍓氶悾閬嶆煕閹惧磭肖闁汇倕妫濆鍫曞灳閸欏鍋?
	if redeemCode.Type == RedeemTypeSubscription && redeemCode.GroupID == nil {
		return nil, infraerrors.BadRequest("REDEEM_CODE_INVALID", "invalid subscription redeem code: missing group_id")
	}

	// 闂佸吋鍎抽崲鑼躲亹閸ヮ剚鍋ㄩ柕濠忕畱閻撴洖菐閸ワ絽澧插ù?
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	// 婵炶揪缍€濞夋洟寮妶澶婃瀬闁绘鐗嗙粊锕傚箹鐎涙ɑ灏紒銊ｅ姂瀹曟繈鍨鹃懠顒傤啍闁荤姴娲ｇ粈渚€宕㈤鍕闁靛繈鍨婚崹鎶芥煛瀹ュ懏绌挎い鏂挎处缁嬪鎯旈姀锛勭К闂佺儵鏅涢敃銈堛亹閸岀偛缁╅柛鎾楀嫮鏆犻梺鍛婎焽閸犲酣鎮哄▎鎾崇畱?
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// 闁诲繐绻愬Λ鏃傝姳閵娾晛绀夐柍鈺佸暞閺夊綊鏌?context闂佹寧绋戞總鏃€绻?repository 闂佸搫鍊介～澶屾兜閸洘鍤勯柦妯侯樈濡绢喖霉閿濆牊纭堕柡浣靛€濆畷銉т沪閼测晩浼囨繛瀛樼矊椤戝懏鎱?
	txCtx := dbent.NewTxContext(ctx, tx)

	// 闂侀潧妫欓崝鏇㈠矗瑜旈弻銊╊敊鐞涒€充壕闁瑰瓨绻傜敮銉╂煛瀹ュ懏绌挎い鏂挎喘瀹曟骞嬮悙鎻掔哎闂佹椿鍠曞鎺旀嫻閻斿鍟呴棅顐幖閳诲繘鏌ｉ姀鈺冨帨缂佽鲸绻勫☉鐢割敊閼姐倗顔斿Δ鐘靛仩濞夋稖銇愰崒婧惧亾閻熼偊妲搁柛?
	// 闂佸憡宸婚弲婵嬪极閵堝鏋侀柣妤€鐗嗙粊锕傚箹鐎涙ɑ灏紒鐘冲閹叉挳宕掗悙瀛樻殎闂佹寧绋戝﹢绗籈RE status = 'unused'闂佹寧绋戦ˇ顔炬崲濮樿鲸瀚氬ù锝囶焾閺傃囨倵濞戞瑥濮堥柍?
	if err := s.redeemRepo.Use(txCtx, redeemCode.ID, userID); err != nil {
		if errors.Is(err, ErrRedeemCodeNotFound) || errors.Is(err, ErrRedeemCodeUsed) {
			return nil, ErrRedeemCodeUsed
		}
		return nil, fmt.Errorf("mark code as used: %w", err)
	}

	// 闂佸湱鐟抽崱鈺傛杸闂佺绻戦崹璺虹暦閺屻儲鐒婚柡鍕箳鐢棝鏌ㄥ☉妯煎闁告﹩鍓熼獮鎴﹀閵忋垹鐏ｉ悗瑙勭摃鐏忣亪锝為敃鍌涚叆濞达絽鎽滈弳浼存煥濞戞ɑ婀版い鎺撶箞瀵喚鎹勯崫鍕唹闁诲海鎳撻ˇ顖炲矗韫囨稑绠肩€广儱瀚粙濠囨煥?
	switch redeemCode.Type {
	case RedeemTypeBalance:
		amount := redeemCode.Value
		// 闁荤姵鍔楅崰鎰板汲閻斿摜鈻旀繛宸簴閸嬫捇鍩€椤掆偓閳诲酣鎮欓鈧埛蹇涙煕閹存繈鐛滅紒杈ㄧ箖閹峰懎鈻庨幙鍐╂緭闂佸搫鐗冮崑鎾趁归敐鍛嚬閻?0
		if amount < 0 && user.Balance+amount < 0 {
			amount = -user.Balance
		}
		if err := s.userRepo.UpdateBalance(txCtx, userID, amount); err != nil {
			return nil, fmt.Errorf("update user balance: %w", err)
		}

	case RedeemTypeConcurrency:
		delta := int(redeemCode.Value)
		// 闁荤姵鍔楅崰鎰板汲閻斿摜鈻旀繛宸簴閸嬫捇鍩€椤掆偓閳诲酣鎮欓鈧埛蹇涙煕閹存繈鐛滅紒杈ㄧ箞閻涱喚鎹勯崫鍕矝闂佽桨鑳舵晶妤€銆掗懜鍨閹煎瓨绻嗙粈?0
		if delta < 0 && user.Concurrency+delta < 0 {
			delta = -user.Concurrency
		}
		if err := s.userRepo.UpdateConcurrency(txCtx, userID, delta); err != nil {
			return nil, fmt.Errorf("update user concurrency: %w", err)
		}

	case RedeemTypeSubscription:
		validityDays := redeemCode.ValidityDays
		if validityDays < 0 {
			// 闁荤姵鍔楅崰鎰板汲閻旂绶為柍鍝勫€瑰▓鍫曟煥濞戞瑦鐨戠紓鍌氼樀閹矂顢橀埥鍡楁倕闂傚倸鍟幏鎴犳濠靛绀勯煫鍥ㄦ尭閻?0 闂佸憡甯楅悷銉ㄣ亹閸パ€妲堥柛顐秵閸氬倿姊?
			if err := s.reduceOrCancelSubscription(txCtx, userID, *redeemCode.GroupID, -validityDays, redeemCode.Code); err != nil {
				return nil, fmt.Errorf("reduce or cancel subscription: %w", err)
			}
		} else {
			if validityDays == 0 {
				validityDays = 30
			}
			_, _, err := s.subscriptionService.AssignOrExtendSubscription(txCtx, &AssignSubscriptionInput{
				UserID:       userID,
				GroupID:      *redeemCode.GroupID,
				ValidityDays: validityDays,
				AssignedBy:   0, // 缂備緡鍨靛畷鐢靛垝濞差亜绀嗛柛鈩冪☉鐢?
				Notes:        fmt.Sprintf("redeemed subscription code %s", redeemCode.Code),
			})
			if err != nil {
				return nil, fmt.Errorf("assign or extend subscription: %w", err)
			}
		}

	default:
		return nil, fmt.Errorf("unsupported redeem type: %s", redeemCode.Type)
	}

	// 闂佸湱绮崝鎺戭潩閿旂晫顩查悗锝庝簻椤?
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	// Transaction committed; invalidate caches.
	s.invalidateRedeemCaches(ctx, userID, redeemCode)

	// Reload the updated redeem code.
	redeemCode, err = s.redeemRepo.GetByID(ctx, redeemCode.ID)
	if err != nil {
		return nil, fmt.Errorf("get updated redeem code: %w", err)
	}

	return redeemCode, nil
}

// invalidateRedeemCaches 婵犮垺鍎兼ご鎼佸疾閵夆晛绀傞柟瀵稿仜鎼村﹪鏌ｉ埡鍐剧劸闁告鍥ㄥ剭闁告洦鍘炬径鍕倵?
func (s *RedeemService) invalidateRedeemCaches(ctx context.Context, userID int64, redeemCode *RedeemCode) {
	switch redeemCode.Type {
	case RedeemTypeBalance:
		if s.authCacheInvalidator != nil {
			s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
		}
		if s.billingCacheService == nil {
			return
		}
		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.billingCacheService.InvalidateUserBalance(cacheCtx, userID)
		}()
	case RedeemTypeConcurrency:
		if s.authCacheInvalidator != nil {
			s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
		}
		if s.billingCacheService == nil {
			return
		}
	case RedeemTypeSubscription:
		if s.authCacheInvalidator != nil {
			s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
		}
		if s.billingCacheService == nil {
			return
		}
		if redeemCode.GroupID != nil {
			groupID := *redeemCode.GroupID
			go func() {
				cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = s.billingCacheService.InvalidateSubscription(cacheCtx, userID, groupID)
			}()
		}
	}
}

// GetByID 闂佸搫绉烽～澶婄暤娑擃搳闂佸吋鍎抽崲鑼躲亹閸ヮ剙绀傞柟瀵稿仜鎼村﹪鏌?
func (s *RedeemService) GetByID(ctx context.Context, id int64) (*RedeemCode, error) {
	code, err := s.redeemRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get redeem code: %w", err)
	}
	return code, nil
}

// GetByCode 闂佸搫绉烽～澶婄暤娑撳攷de闂佸吋鍎抽崲鑼躲亹閸ヮ剙绀傞柟瀵稿仜鎼村﹪鏌?
func (s *RedeemService) GetByCode(ctx context.Context, code string) (*RedeemCode, error) {
	redeemCode, err := s.redeemRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("get redeem code: %w", err)
	}
	return redeemCode, nil
}

// List 闂佸吋鍎抽崲鑼躲亹閸ヮ剙绀傞柟瀵稿仜鎼村﹪鏌ｉ鑽ゎ槮闁割煈浜為幃浼粹€﹂幒鏃傤槱缂備胶濯寸槐鏇㈠箖婵犲洤宸濇俊顖氭惈椤娀鏌ら搹顐ｇ伄缂?
func (s *RedeemService) List(ctx context.Context, params pagination.PaginationParams) ([]RedeemCode, *pagination.PaginationResult, error) {
	codes, pagination, err := s.redeemRepo.List(ctx, params)
	if err != nil {
		return nil, nil, fmt.Errorf("list redeem codes: %w", err)
	}
	return codes, pagination, nil
}

// Delete 闂佸憡甯炴繛鈧繛鍛叄瀹曟骞嬮悙鎻掔哎闂佹椿鍠曠欢銈囨濞嗘垹涓嶉柨娑樺閸婄偤鏌涘☉娅亝鎱ㄥ☉銏″殑闁哄倶鍊楃粈?
func (s *RedeemService) Delete(ctx context.Context, id int64) error {
	// 濠碘槅鍋€閸嬫捇鏌＄仦璇插姎闁告﹩鍓熼獮鎴﹀閵忋垹鐏ｉ梺鍝勫閸ㄤ即骞嗘担琛″亾濞戞顏勶耿?
	code, err := s.redeemRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get redeem code: %w", err)
	}

	// 婵炴垶鎸哥粔鎾储閹寸姵濯肩紒瀣仢閻忊晠姊婚崟鈺佲偓鏇㈠礄閳╁啯濯撮悹鎭掑妽閺嗗繘鏌ｉ妸銉ヮ仼闁告﹩鍓熼獮鎴﹀閵忋垹鐏?
	if code.IsUsed() {
		return infraerrors.Conflict("REDEEM_CODE_DELETE_USED", "cannot delete used redeem code")
	}

	if err := s.redeemRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete redeem code: %w", err)
	}

	return nil
}

// GetStats 闂佸吋鍎抽崲鑼躲亹閸ヮ剙绀傞柟瀵稿仜鎼村﹪鏌ｉ鏄忓厡缂侇喛娅ｉ幏瀣灳闊厾绠氶梺?
func (s *RedeemService) GetStats(ctx context.Context) (map[string]any, error) {
	// TODO: 闁诲骸婀遍崑鐔肩嵁閸モ晝纾奸柣鏃€妞块崥鈧梻渚囧亝濡叉帞娆?
	// 缂傚倷鑳堕崰鏇㈩敇閹间礁瀚夋い蹇撳暙閳诲繘鏌ｉ～顒€濡搁柍褜鍏涚粈渚€宕欓埄鍐╁閻犳亽鍔嶉弳蹇涙煟閵娿儱顏╅柛姗嗗墴楠炴垿濮€閵忋垹鐏ｉ梺杞版閸楁娊宕?
	// 缂傚倷鑳堕崰鏇㈩敇閹间礁绠戝┑鐘崇濡椼劑鏌涙繝鍛汗闁?

	stats := map[string]any{
		"total_codes":  0,
		"unused_codes": 0,
		"used_codes":   0,
		"total_value":  0.0,
	}

	return stats, nil
}

// GetUserHistory 闂佸吋鍎抽崲鑼躲亹閸ヮ剚鍋ㄩ柕濠忕畱閻撴洟鏌ｉ妸銉ヮ仼闁告﹩鍓熼獮鎴﹀閻樻彃娼戦梺?
func (s *RedeemService) GetUserHistory(ctx context.Context, userID int64, limit int) ([]RedeemCode, error) {
	codes, err := s.redeemRepo.ListByUser(ctx, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("get user redeem history: %w", err)
	}
	return codes, nil
}

// reduceOrCancelSubscription 缂傚倸鍊甸弲婵嬫偂椤撶姵濯奸柕蹇嬪€栭～澶婎熆閸ㄦ稒娅嗛柡鍡欏枛閺佸秶浠﹂挊澶嗘寘婵炶揪绲鹃悷銉╁Φ婢舵劕鏋?<= 0 闂佸搫鍟冲▔娑溿亹閸パ€妲堥柛顐秵閸氬倿姊?
func (s *RedeemService) reduceOrCancelSubscription(ctx context.Context, userID, groupID int64, reduceDays int, code string) error {
	sub, err := s.subscriptionService.userSubRepo.GetByUserIDAndGroupID(ctx, userID, groupID)
	if err != nil {
		return ErrSubscriptionNotFound
	}

	now := time.Now()
	remaining := int(sub.ExpiresAt.Sub(now).Hours() / 24)
	if remaining < 0 {
		remaining = 0
	}

	notes := fmt.Sprintf("redeemed code %s reduced subscription by %d days", code, reduceDays)

	if remaining <= reduceDays {
		// 闂佸憡鎸撮弲娆戠礊閹寸偛绶為柍鍝勫€瑰▓璺衡槈閹惧磭啸闁告劕顭烽弫宥呯暆閳ь剙煤閸ф绠抽柕澶堝劚缁叉寧绻涢幋婵堝ⅲ妞ゆ挸缍婂?
		if err := s.subscriptionService.userSubRepo.UpdateStatus(ctx, sub.ID, SubscriptionStatusExpired); err != nil {
			return fmt.Errorf("cancel subscription: %w", err)
		}
		// 闁荤姳绀佹晶浠嬫偪閸℃ɑ浜ら柛銉ｅ妽閸╁倿鏌￠崘銊у煟婵☆偅瀵х粙澶愬传閸曨厾歇闂佸憡鎸哥粔闈涱渻閸岀偞鈷?
		if err := s.subscriptionService.userSubRepo.ExtendExpiry(ctx, sub.ID, now); err != nil {
			return fmt.Errorf("set subscription expiry: %w", err)
		}
	} else {
		// 缂傚倸鍊甸弲婵嬫偂椤撶喎绶為柍鍝勫€瑰▓?
		newExpiresAt := sub.ExpiresAt.AddDate(0, 0, -reduceDays)
		if err := s.subscriptionService.userSubRepo.ExtendExpiry(ctx, sub.ID, newExpiresAt); err != nil {
			return fmt.Errorf("reduce subscription: %w", err)
		}
	}

	// 闁哄鏅炲Λ鍕叏閻愭潙绶為柛銉ｅ妽閺?
	newNotes := sub.Notes
	if newNotes != "" {
		newNotes += "\n"
	}
	newNotes += notes
	if err := s.subscriptionService.userSubRepo.UpdateNotes(ctx, sub.ID, newNotes); err != nil {
		return fmt.Errorf("update subscription notes: %w", err)
	}

	// 婵犮垺鍎兼ご鎼佸疾閵壯呯＝闁规儳纾幗?
	s.subscriptionService.InvalidateSubCache(userID, groupID)

	return nil
}
