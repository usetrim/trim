package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/usetrim/trim/server/internal/account"
	"github.com/usetrim/trim/server/internal/authsettings"
	"github.com/usetrim/trim/server/internal/avatar"
	"github.com/usetrim/trim/server/internal/billing"
	"github.com/usetrim/trim/server/internal/billing/paddleapi"
	"github.com/usetrim/trim/server/internal/billingsettings"
	"github.com/usetrim/trim/server/internal/config"
	"github.com/usetrim/trim/server/internal/dbpool"
	"github.com/usetrim/trim/server/internal/emaildenylist"
	"github.com/usetrim/trim/server/internal/events"
	"github.com/usetrim/trim/server/internal/idechrome"
	"github.com/usetrim/trim/server/internal/mailer"
	"github.com/usetrim/trim/server/internal/middleware"
	"github.com/usetrim/trim/server/internal/pagination"
	"github.com/usetrim/trim/server/internal/plans"
	"github.com/usetrim/trim/server/internal/platformadmin"
	"github.com/usetrim/trim/server/internal/proxy"
	"github.com/usetrim/trim/server/internal/receipts"
	"github.com/usetrim/trim/server/internal/subscriptions"
	"github.com/usetrim/trim/server/internal/telemetry"
	"github.com/usetrim/trim/server/internal/workspaces"
)

func main() {
	_ = godotenv.Load()
	// Scratch images have no wget/curl; compose/K8s HEALTHCHECK uses this binary flag.
	if len(os.Args) > 1 && os.Args[1] == "-readyz" {
		os.Exit(runReadyzProbe())
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()

	db, err := dbpool.Open(ctx, cfg.DatabaseURL, cfg.PGMaxConns, cfg.PGMinConns)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	log.Println("connected to postgres")

	readDB := db
	if strings.TrimSpace(cfg.DatabaseReadURL) != "" {
		readDB, err = dbpool.Open(ctx, cfg.DatabaseReadURL, cfg.PGReadMaxConns, cfg.PGReadMinConns)
		if err != nil {
			log.Fatalf("database read: %v", err)
		}
		log.Println("connected to postgres read pool")
	} else {
		log.Println("DATABASE_READ_URL unset; reads use primary")
	}

	// Fail closed in every deployment mode: chrome must be DB-driven after warm.
	// Continuing on seed failure would serve invent English from builtins.
	if err := subscriptions.SeedAndRefreshSiteMessages(ctx, db); err != nil {
		log.Fatalf("site_messages seed/refresh: %v", err)
	}
	log.Println("site_messages cache warmed")

	if len(cfg.AllowedAuthProviders) > 0 {
		if err := authsettings.SyncAllowedProviders(ctx, db, cfg.AllowedAuthProviders); err != nil {
			// Cloud fail closed: signup trigger reads auth_settings; stale allow-list is unsafe.
			if cfg.DeploymentMode == "cloud" {
				log.Fatalf("auth_settings sync: %v", err)
			}
			log.Printf("auth_settings sync: %v (self-host continuing; signup may use previous DB allow-list)", err)
		} else {
			log.Printf("auth_settings synced providers: %s", strings.Join(cfg.AllowedAuthProviders, ","))
		}
	}

	if err := middleware.InitGeoLite(cfg.GeoLiteASNMMDBPath, cfg.GeoLiteAnonymousMMDBPath); err != nil {
		log.Printf("geolite: %v (continuing with CF + ipwho.is only)", err)
	} else if cfg.GeoLiteASNMMDBPath != "" || cfg.GeoLiteAnonymousMMDBPath != "" {
		log.Println("geolite MMDB loaded for offline IP risk")
	}

	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis url: %v", err)
	}
	rdb := redis.NewClient(opt)
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping: %v", err)
	}
	log.Println("connected to redis")

	proxyServer := proxy.NewServer(cfg, db)
	eventsHandler := events.NewHandler(events.HandlerOpts{
		DB:               db,
		ReadDB:           readDB,
		Redis:            rdb,
		InsertTimeoutSec: cfg.EventInsertTimeoutSec,
		QueueSize:        cfg.EventInsertQueueSize,
		Workers:          cfg.EventInsertWorkers,
		OutboxPollSec:    cfg.EventOutboxPollSec,
		OutboxBatch:      cfg.EventOutboxBatch,
	})
	proxyServer.SetEventSink(func(userID, requestID, model string, before, after int, latencyMs float64, status, mode, errorCode string) {
		eventsHandler.RecordAsync(userID, requestID, model, before, after, int(latencyMs+0.5), status, mode, errorCode)
	})
	authMW := middleware.NewAuthQuota(db, rdb, cfg)
	authMW.Metrics = proxyServer.Metrics
	fraudMW := middleware.NewAntiFraud(db, rdb, cfg)
	fraudMW.OnHMACFail = func() {
		if proxyServer.Metrics != nil {
			proxyServer.Metrics.IncHMACFailure()
		}
	}
	var paddleClient *paddleapi.Client
	if cfg.DeploymentMode == "cloud" {
		paddleClient, err = paddleapi.New(
			cfg.PaddleAPIKey,
			cfg.PaddleEnv,
			cfg.PaddleHTTPTimeoutSec,
			cfg.PaddleListMaxPages,
			cfg.PaddleListPerPage,
		)
		if err != nil {
			log.Fatalf("paddle api: %v", err)
		}
	}
	paddleHandler := billing.NewHandler(db, rdb, cfg.PaddleWebhookSecret, paddleClient, cfg.WebhookProcessTimeoutSec, cfg.WebhookQueueSize, cfg.WebhookWorkers)
	plansHandler := plans.NewHandler(db, readDB, paddleClient, rdb, cfg.EnterpriseNotifyURL, cfg.EnterpriseNotifyTimeoutSec)
	receiptsHandler := receipts.NewHandler(db, readDB, paddleClient)
	accountHandler := account.NewHandler(db, readDB)
	avatarHandler := avatar.NewHandler(db, cfg.CloudinaryHTTPTimeoutSec)
	telemetryHandler := telemetry.NewHandler(rdb, cfg.RedisTelemetryTTLSec)
	workspacesHandler := workspaces.NewHandler(db, readDB, cfg.AppPublicURL, mailer.SMTPConfig{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		Username: cfg.SMTPUsername,
		Password: cfg.SMTPPassword,
		From:     cfg.SMTPFrom,
	})
	adminHandler := platformadmin.NewHandler(db, readDB, rdb, cfg, paddleHandler, paddleClient, mailer.SMTPConfig{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		Username: cfg.SMTPUsername,
		Password: cfg.SMTPPassword,
		From:     cfg.SMTPFrom,
	})
	if err := adminHandler.EnsureTrimEventsPartitions(ctx); err != nil {
		log.Fatalf("trim_events partitions: %v", err)
	}
	if _, err := adminHandler.PartitionEnsureSec(ctx); err != nil {
		log.Fatalf("trim_events partition ensure interval: %v", err)
	}
	log.Println("trim_events month partitions ensured")
	if ovr, err := middleware.LoadRuntimeOverridesFromDB(ctx, db); err == nil {
		_ = middleware.PublishRuntimeOverrides(ctx, rdb, ovr)
	} else {
		log.Printf("admin runtime overrides: %v (using env Config only)", err)
	}
	subsService := plansHandler.Subs

	subCtx, subCancel := context.WithCancel(context.Background())
	defer subCancel()
	go subscriptions.SubscribeSiteMessagesReload(subCtx, db, rdb, cfg.SiteMessagesReloadSec)
	go adminHandler.RunPartitionEnsureLoop(subCtx)

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: cfg.CORSOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-Hardware-UUID", "X-Trim-Agent-Id", "X-Client-Version", "X-Request-Timestamp", "X-Trim-Signature", "X-Workspace-Id", "X-Trim-Step-Up"},
		// PDF downloads need filename headers readable cross-origin (not CORS-safelisted).
		ExposedHeaders: []string{
			"X-Trim-Admin-StepUp-Policy",
			"Content-Disposition",
			"Content-Type",
			"X-Trim-Download-Filename",
		},
		AllowCredentials: true,
		MaxAge:           cfg.CORSMaxAgeSec,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":    "ok",
			"service":   "trim-api",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		type check struct {
			PostgresWrite string `json:"postgres_write"`
			PostgresRead  string `json:"postgres_read"`
			Redis         string `json:"redis"`
		}
		out := map[string]any{
			"service":   "trim-api",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		c := check{PostgresWrite: "ok", PostgresRead: "ok", Redis: "ok"}
		ready := true
		if err := db.Ping(r.Context()); err != nil {
			c.PostgresWrite = "error"
			ready = false
		}
		if err := readDB.Ping(r.Context()); err != nil {
			c.PostgresRead = "error"
			ready = false
		}
		if err := rdb.Ping(r.Context()).Err(); err != nil {
			c.Redis = "error"
			ready = false
		}
		out["checks"] = c
		if !ready {
			out["status"] = "not_ready"
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			out["status"] = "ready"
		}
		_ = json.NewEncoder(w).Encode(out)
	})

	r.Get("/metrics", proxyServer.Metrics.Handler())

	r.Post("/webhooks/paddle", paddleHandler.ServeHTTP)

	// Layer 8 decoy routes (trip canary bans via fraud middleware; no auth required).
	r.Group(func(decoy chi.Router) {
		decoy.Use(fraudMW.Protect)
		notFound := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": subscriptions.MessageForCode("CANARY_NOT_FOUND"),
			})
		}
		decoy.HandleFunc("/api/v1/internal/free-credits", notFound)
		decoy.HandleFunc("/api/v1/internal/grant-credits", notFound)
		decoy.HandleFunc("/api/v1/admin/bypass-quota", notFound)
		decoy.HandleFunc("/v1/internal/free-credits", notFound)
	})

	if cfg.CronSecret != "" && subsService != nil {
		r.Post("/internal/expire-subscriptions", func(w http.ResponseWriter, r *http.Request) {
			secret := strings.TrimSpace(r.Header.Get("X-Cron-Secret"))
			if secret == "" || secret != cfg.CronSecret {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": subscriptions.MessageForCode("CRON_UNAUTHORIZED"),
				})
				return
			}
			limit := cfg.ExpireBatchDefaultLimit
			if limit < 1 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": subscriptions.MessageForCode("EXPIRE_BATCH_LIMIT_MISSING"),
				})
				return
			}
			if v := r.URL.Query().Get("limit"); v != "" {
				if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 2000 {
					limit = n
				}
			}
			processed, changed, err := subsService.ExpireDueBatch(r.Context(), limit)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": subscriptions.MessageForCode("EXPIRE_BATCH_FAILED"),
				})
				return
			}
			// Clear Redis quota caches for changed users is best-effort via next request ExpireIfNeeded.
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]int{
				"processed": processed,
				"changed":   changed,
			})
		})
	}

	if cfg.CronSecret != "" {
		r.Post("/internal/retention-purge", func(w http.ResponseWriter, r *http.Request) {
			secret := strings.TrimSpace(r.Header.Get("X-Cron-Secret"))
			if secret == "" || secret != cfg.CronSecret {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
				return
			}
			result, err := adminHandler.RunRetentionPurge(r.Context())
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				code := err.Error()
				status := http.StatusInternalServerError
				if code == "ADMIN_RETENTION_TTL_REQUIRED" {
					status = http.StatusConflict
				}
				if code == "ADMIN_EVENTS_PARTITION_AHEAD_MISSING" || code == "ADMIN_EVENTS_PARTITION_ENSURE_FAILED" {
					status = http.StatusServiceUnavailable
				}
				w.WriteHeader(status)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": code})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(result)
		})
	}

	r.Route("/api/v1", func(api chi.Router) {
		// Postgres IP/ASN denylist on ALL /api/v1 traffic (public + authenticated).
		// Protect still re-checks on auth routes (Redis-cached; cheap).
		api.Use(fraudMW.Denylist)

		api.With(authMW.OptionalAuth).Get("/public/plans", plansHandler.ListPublic)
		api.Get("/public/auth-providers", authsettings.PublicProvidersHandler(readDB, cfg.CompanySupportEmail))
		api.Get("/public/email-policy", emaildenylist.PublicCheckHandler(readDB))
		api.Get("/public/ide-chrome", idechrome.PublicChromeHandler(readDB))
		api.Get("/public/workspace-invites/{token}", workspacesHandler.PreviewInvite)
		api.Post("/public/install-hit", adminHandler.InstallHit)
		api.Post("/telemetry", telemetryHandler.Ingest)

		api.Group(func(prot chi.Router) {
			prot.Use(fraudMW.Protect)
			prot.Use(authMW.Secure)

			// Instance operator console (apps/admin): available in cloud and self.
			// Open-core rule: self-host manages THIS instance only. Cloud still requires
			// ADMIN_ALLOWED_ORIGINS / Gate. Never a Trim-Inc remote backdoor into other installs.
			prot.Route("/admin", func(ar chi.Router) {
				ar.Use(adminHandler.Gate)
				adminHandler.Mount(ar)
			})

			prot.Get("/me/quota", handleQuota(db, rdb, cfg.AppPublicURL))
			prot.Get("/me/stats", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(proxyServer.Stats())
			})
			prot.Get("/me/events", handleEvents(readDB))
			prot.Get("/me/events/stats", eventsHandler.Stats)
			prot.Post("/me/events", eventsHandler.Ingest)
			prot.Delete("/me/events", eventsHandler.BulkDelete)
			prot.Post("/me/avatar/sync", avatarHandler.Sync)
			prot.Get("/me/api-keys", accountHandler.ListAPIKeys)
			prot.Post("/me/api-keys", accountHandler.CreateAPIKey)
			prot.Delete("/me/api-keys/{keyId}", accountHandler.RevokeAPIKey)
			prot.Get("/me/api-keys/{keyId}/devices", accountHandler.ListAPIKeyDevices)
			prot.Post("/me/api-keys/{keyId}/devices", accountHandler.RegisterAPIKeyDevice)
			prot.Delete("/me/api-keys/{keyId}/devices/{hardwareUUID}", accountHandler.DeleteAPIKeyDevice)
			prot.Get("/me/notifications", accountHandler.ListNotifications)
			prot.Get("/me/notifications/unread-count", accountHandler.UnreadNotificationCount)
			prot.Post("/me/notifications/read-all", accountHandler.MarkAllNotificationsRead)
			prot.Post("/me/notifications/{id}/read", accountHandler.MarkNotificationRead)
			prot.Get("/me/preferences", accountHandler.GetPreferences)
			prot.Patch("/me/preferences", accountHandler.PatchPreferences)
			prot.Delete("/me/account", accountHandler.DeleteAccount)
			prot.Get("/billing/subscription", plansHandler.GetSubscriptionStatus)
			prot.Post("/billing/checkout-preview", plansHandler.PreviewCheckoutSession)
			prot.Post("/billing/checkout-session", plansHandler.CreateCheckoutSession)
			prot.Post("/billing/portal-session", plansHandler.CreatePortalSession)
			prot.Post("/billing/enterprise-inquiry", plansHandler.CreateEnterpriseInquiry)
			prot.Get("/me/enterprise-inquiries", plansHandler.ListMeEnterpriseInquiries)
			prot.Delete("/me/enterprise-inquiries", plansHandler.DeleteMeEnterpriseInquiries)
			prot.Get("/billing/receipts", receiptsHandler.List(cfg.CompanyLegalName, cfg.CompanySupportEmail))
			prot.Get("/billing/receipts/{receiptId}", receiptsHandler.Get(receipts.SellerLetterhead{
				LegalName:    cfg.CompanyLegalName,
				SupportEmail: cfg.CompanySupportEmail,
				LogoURL:      cfg.CompanyLogoURL,
				AddressLine1: cfg.CompanyAddressLine1,
				AddressLine2: cfg.CompanyAddressLine2,
				City:         cfg.CompanyCity,
				Region:       cfg.CompanyRegion,
				PostalCode:   cfg.CompanyPostalCode,
				Country:      cfg.CompanyCountry,
				VATID:        cfg.CompanyVATID,
				Registration: cfg.CompanyRegistration,
			}))
			prot.Get("/billing/receipts/{receiptId}/pdf", receiptsHandler.DownloadPDF(receipts.SellerLetterhead{
				LegalName:    cfg.CompanyLegalName,
				SupportEmail: cfg.CompanySupportEmail,
				LogoURL:      cfg.CompanyLogoURL,
				AddressLine1: cfg.CompanyAddressLine1,
				AddressLine2: cfg.CompanyAddressLine2,
				City:         cfg.CompanyCity,
				Region:       cfg.CompanyRegion,
				PostalCode:   cfg.CompanyPostalCode,
				Country:      cfg.CompanyCountry,
				VATID:        cfg.CompanyVATID,
				Registration: cfg.CompanyRegistration,
			}))
			prot.Post("/billing/receipts/sync", paddleHandler.SyncReceipts)
			prot.Get("/workspaces", workspacesHandler.ListMine)
			prot.Post("/workspaces", workspacesHandler.Create)
			prot.Patch("/workspaces/{workspaceId}", workspacesHandler.Rename)
			prot.Delete("/workspaces/{workspaceId}", workspacesHandler.Delete)
			prot.Get("/workspaces/{workspaceId}/members", workspacesHandler.ListMembers)
			prot.Get("/workspaces/{workspaceId}/invites", workspacesHandler.ListInvites)
			prot.Post("/workspaces/{workspaceId}/invite", workspacesHandler.Invite)
			prot.Delete("/workspaces/{workspaceId}/invites/{inviteId}", workspacesHandler.RevokeInvite)
			prot.Patch("/workspaces/{workspaceId}/members/{memberId}", workspacesHandler.UpdateMemberRole)
			prot.Delete("/workspaces/{workspaceId}/members/{memberId}", workspacesHandler.RemoveMember)
			prot.Post("/workspace-invites/{token}/accept", workspacesHandler.AcceptInvite)
		})
	})

	// Cloud mode: LLM proxy requires auth + quota. Self-hosted mode is open local.
	if cfg.DeploymentMode == "cloud" {
		r.Group(func(pr chi.Router) {
			pr.Use(fraudMW.Protect)
			pr.Use(authMW.Secure)
			pr.Mount("/", proxyServer.Handler())
		})
	} else {
		r.Mount("/", proxyServer.Handler())
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: time.Duration(cfg.HTTPReadHeaderTimeoutSec) * time.Second,
	}

	go func() {
		region := strings.TrimSpace(os.Getenv("TRIM_REGION_LABEL"))
		if region != "" {
			log.Printf("Trim API listening on :%s (mode=%s region=%s)", cfg.Port, cfg.DeploymentMode, region)
		} else {
			log.Printf("Trim API listening on :%s (mode=%s)", cfg.Port, cfg.DeploymentMode)
		}
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.HTTPShutdownTimeoutSec)*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	db.Close()
	_ = rdb.Close()
	log.Println("Trim API stopped")
}

func handleQuota(db *pgxpool.Pool, rdb *redis.Client, appPublicURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.UserIDFromContext(r.Context())
		if userID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": subscriptions.MessageForCode("WS_UNAUTHORIZED")})
			return
		}

		type quota struct {
			PlanTier           string `json:"plan_tier"`
			Limit              int    `json:"monthly_credit_limit"`
			Used               int    `json:"monthly_credit_used"`
			Topup              int    `json:"purchased_topup_credits"`
			Remaining          int    `json:"remaining"`
			Unlimited          bool   `json:"unlimited"`
			UnlimitedLabel     string `json:"unlimited_label,omitempty"`
			Code               string `json:"code,omitempty"`
			Error              string `json:"error,omitempty"`
			TierUpgrade        string `json:"tier_upgrade,omitempty"`
			ExhaustedTitle     string `json:"exhausted_title,omitempty"`
			ExhaustedBody      string `json:"exhausted_body,omitempty"`
			UpgradeActionLabel string `json:"upgrade_action_label,omitempty"`
		}

		var q quota
		err := db.QueryRow(r.Context(), `
			select uq.plan_tier, uq.monthly_credit_limit, uq.monthly_credit_used, uq.purchased_topup_credits,
			       pc.unlimited
			from public.user_quotas uq
			inner join public.plan_catalog pc on pc.id = uq.plan_tier and pc.is_active = true
			where uq.user_id = $1
		`, userID).Scan(&q.PlanTier, &q.Limit, &q.Used, &q.Topup, &q.Unlimited)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": subscriptions.MessageForCode("QUOTA_NOT_FOUND"),
			})
			return
		}

		catalogUnlimited := q.Unlimited
		if rdb != nil {
			val, err := rdb.HGetAll(r.Context(), "user_quota:"+userID).Result()
			if err == nil && len(val) > 0 {
				if v, ok := val["used"]; ok {
					if n, e := strconv.Atoi(v); e == nil {
						q.Used = n
					}
				}
				if v, ok := val["topup"]; ok {
					if n, e := strconv.Atoi(v); e == nil {
						q.Topup = n
					}
				}
				if v, ok := val["limit"]; ok {
					if n, e := strconv.Atoi(v); e == nil {
						q.Limit = n
					}
				}
				// Never trust Redis for unlimited: plan_catalog is source of truth.
				// Stale unlimited=0 after webhook entitlement / catalog repair caused false "exhausted".
				if v, ok := val["unlimited"]; ok && (v == "0" || v == "1") && (v == "1") != catalogUnlimited {
					_ = rdb.HSet(r.Context(), "user_quota:"+userID, "unlimited", map[bool]string{true: "1", false: "0"}[catalogUnlimited]).Err()
				}
			}
		}
		q.Unlimited = catalogUnlimited

		q.Remaining = (q.Limit - q.Used) + q.Topup
		upgradePath := strings.TrimSpace(subscriptions.MessageForCode("APP_PATH_UPGRADE"))
		appPublic := strings.TrimRight(strings.TrimSpace(appPublicURL), "/")
		if appPublic != "" && upgradePath != "" {
			q.TierUpgrade = appPublic + upgradePath
		}
		if !q.Unlimited && q.Remaining <= 0 {
			q.Code = "QUOTA_EXHAUSTED"
			q.Error = subscriptions.MessageForCode("QUOTA_EXHAUSTED")
			q.ExhaustedTitle = subscriptions.MessageForCode("DASHBOARD_QUOTA_EXHAUSTED_TITLE")
			q.ExhaustedBody = subscriptions.MessageForCode("DASHBOARD_QUOTA_EXHAUSTED_BODY")
			q.UpgradeActionLabel = subscriptions.MessageForCode("IDE_QUOTA_UPGRADE_ACTION")
		}
		if q.Unlimited {
			q.UnlimitedLabel = subscriptions.MessageForCode("DASHBOARD_QUOTA_UNLIMITED_LABEL")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(q)
	}
}

func handleEvents(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.UserIDFromContext(r.Context())
		if userID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": subscriptions.MessageForCode("WS_UNAUTHORIZED")})
			return
		}
		maxLimit, err := billingsettings.MaxPageSize(r.Context(), db)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": subscriptions.MessageForCode(err.Error())})
			return
		}
		skipCap, err := billingsettings.SkipToMaxPages(r.Context(), db)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": subscriptions.MessageForCode(err.Error())})
			return
		}
		params, err := pagination.Parse(r, maxLimit)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": subscriptions.MessageForCode(err.Error())})
			return
		}

		fromDay, toDay, dateErr := parseOptionalUTCDateRange(r)
		if dateErr != "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": subscriptions.MessageForCode(dateErr)})
			return
		}

		where := `user_id = $1`
		args := []interface{}{userID}
		argN := 2
		if fromDay != "" {
			where += fmt.Sprintf(` and created_at >= ($%d::date at time zone 'utc')`, argN)
			args = append(args, fromDay)
			argN++
		}
		if toDay != "" {
			where += fmt.Sprintf(` and created_at < (($%d::date + interval '1 day') at time zone 'utc')`, argN)
			args = append(args, toDay)
			argN++
		}
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if len(q) > 200 {
			q = q[:200]
		}
		if q != "" {
			like := "%" + escapeILikePattern(q) + "%"
			where += fmt.Sprintf(` and (
				coalesce(model, '') ilike $%d escape '\'
				or coalesce(mode, '') ilike $%d escape '\'
				or coalesce(status, '') ilike $%d escape '\'
				or coalesce(request_id, '') ilike $%d escape '\'
			)`, argN, argN, argN, argN)
			args = append(args, like)
			argN++
		}

		var total int
		countSQL := `select count(*) from public.trim_events where ` + where
		if err := db.QueryRow(r.Context(), countSQL, args...).Scan(&total); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": subscriptions.MessageForCode("EVENTS_COUNT_FAILED")})
			return
		}

		listSQL := fmt.Sprintf(`
			select id::text, request_id, model, tokens_before, tokens_after, latency_ms, mode, status, created_at::text
			from public.trim_events
			where %s
			order by created_at desc
			offset $%d limit $%d
		`, where, argN, argN+1)
		listArgs := append(append([]interface{}{}, args...), params.Skip, params.Limit)
		rows, err := db.Query(r.Context(), listSQL, listArgs...)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": subscriptions.MessageForCode("EVENTS_LIST_FAILED")})
			return
		}
		defer rows.Close()

		type eventRow struct {
			ID             string  `json:"id"`
			RequestID      *string `json:"request_id"`
			Model          *string `json:"model"`
			TokensBefore   int     `json:"tokens_before"`
			TokensAfter    int     `json:"tokens_after"`
			LatencyMs      int     `json:"latency_ms"`
			Mode           string  `json:"mode"`
			ModeLabel      string  `json:"mode_label"`
			Status         string  `json:"status"`
			StatusLabel    string  `json:"status_label"`
			CreatedAt      string  `json:"created_at"`
			CreatedAtLabel string  `json:"created_at_label"`
		}
		items := make([]eventRow, 0)
		for rows.Next() {
			var e eventRow
			if err := rows.Scan(&e.ID, &e.RequestID, &e.Model, &e.TokensBefore, &e.TokensAfter, &e.LatencyMs, &e.Mode, &e.Status, &e.CreatedAt); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": subscriptions.MessageForCode("EVENTS_SCAN_FAILED")})
				return
			}
			e.ModeLabel = subscriptions.EventModeLabel(e.Mode)
			e.StatusLabel = subscriptions.EventStatusLabel(e.Status)
			e.CreatedAtLabel = subscriptions.FormatUTCDateTimeFromRFC3339(e.CreatedAt)
			items = append(items, e)
		}

		var dateRangeMonths int
		if err := db.QueryRow(r.Context(), `
			select coalesce(date_range_months, 0)
			from public.billing_settings where id = 'default'
		`).Scan(&dateRangeMonths); err != nil || dateRangeMonths < 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": subscriptions.MessageForCode("BILLING_SETTINGS_UNAVAILABLE")})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"items":                       items,
			"meta":                        pagination.BuildMeta(params, total, skipCap),
			"empty_message":               subscriptions.MessageForCode("EVENTS_EMPTY"),
			"col_when":                    subscriptions.MessageForCode("EVENTS_COL_WHEN"),
			"col_model":                   subscriptions.MessageForCode("EVENTS_COL_MODEL"),
			"col_mode":                    subscriptions.MessageForCode("EVENTS_COL_MODE"),
			"col_tokens":                  subscriptions.MessageForCode("EVENTS_COL_TOKENS"),
			"col_latency":                 subscriptions.MessageForCode("EVENTS_COL_LATENCY"),
			"col_status":                  subscriptions.MessageForCode("EVENTS_COL_STATUS"),
			"col_request_id":              subscriptions.MessageForCode("EVENTS_COL_REQUEST_ID"),
			"col_view":                    subscriptions.MessageForCode("EVENTS_COL_VIEW"),
			"open_action_label":           subscriptions.MessageForCode("EVENTS_OPEN_LABEL"),
			"table_row_actions":           subscriptions.MessageForCode("TABLE_ROW_ACTIONS"),
			"preview_field_description":   subscriptions.MessageForCode("EVENTS_PREVIEW_FIELD_DESC"),
			"tokens_sep":                  subscriptions.MessageForCode("EVENTS_TOKENS_SEP"),
			"latency_unit":                subscriptions.MessageForCode("EVENTS_LATENCY_UNIT"),
			"date_from":                   fromDay,
			"date_to":                     toDay,
			"date_range_placeholder":      subscriptions.MessageForCode("EVENTS_DATE_RANGE_PLACEHOLDER"),
			"date_range_description":      subscriptions.MessageForCode("EVENTS_DATE_RANGE_DESC"),
			"date_range_clear":            subscriptions.MessageForCode("EVENTS_DATE_RANGE_CLEAR"),
			"date_range_apply":            subscriptions.MessageForCode("EVENTS_DATE_RANGE_APPLY"),
			"date_range_months":           dateRangeMonths,
			"search_placeholder":          subscriptions.MessageForCode("EVENTS_SEARCH"),
			"search_description":          subscriptions.MessageForCode("EVENTS_SEARCH_DESC"),
			"q":                           q,
			"table_select_all":            subscriptions.MessageForCode("TABLE_SELECT_ALL"),
			"table_select_row":            subscriptions.MessageForCode("TABLE_SELECT_ROW"),
			"table_selected_fmt":          subscriptions.MessageForCode("TABLE_SELECTED_FMT"),
			"table_bulk_delete":           subscriptions.MessageForCode("TABLE_BULK_DELETE"),
			"table_clear_selection":       subscriptions.MessageForCode("TABLE_CLEAR_SELECTION"),
			"delete_action_label":         subscriptions.MessageForCode("EVENTS_DELETE"),
			"delete_pending_label":        subscriptions.MessageForCode("EVENTS_DELETE_PENDING"),
			"delete_confirm_message":      subscriptions.MessageForCode("EVENTS_DELETE_CONFIRM"),
			"bulk_delete_confirm_message": subscriptions.MessageForCode("EVENTS_BULK_DELETE_CONFIRM"),
		})
	}
}

// escapeILikePattern escapes \, %, and _ for PostgreSQL ILIKE ... ESCAPE '\'.
func escapeILikePattern(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// parseOptionalUTCDateRange reads from/to query params as YYYY-MM-DD (UTC calendar days).
// Empty means no filter. Returns a site_messages code on invalid input.
func parseOptionalUTCDateRange(r *http.Request) (fromDay, toDay string, errCode string) {
	fromDay = strings.TrimSpace(r.URL.Query().Get("from"))
	toDay = strings.TrimSpace(r.URL.Query().Get("to"))
	if fromDay == "" && toDay == "" {
		return "", "", ""
	}
	if fromDay != "" {
		if _, err := time.Parse("2006-01-02", fromDay); err != nil {
			return "", "", "EVENTS_DATE_FROM_INVALID"
		}
	}
	if toDay != "" {
		if _, err := time.Parse("2006-01-02", toDay); err != nil {
			return "", "", "EVENTS_DATE_TO_INVALID"
		}
	}
	if fromDay != "" && toDay != "" && fromDay > toDay {
		return "", "", "EVENTS_DATE_RANGE_ORDER"
	}
	return fromDay, toDay, ""
}

// runReadyzProbe opens Postgres (+ optional read URL) and Redis, pings each, exits 0/1.
// Used by Docker HEALTHCHECK on scratch images (no wget/curl in the container).
func runReadyzProbe() int {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("readyz: config: %v", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.ReadyzTimeoutSec)*time.Second)
	defer cancel()

	db, err := dbpool.Open(ctx, cfg.DatabaseURL, cfg.PGMaxConns, cfg.PGMinConns)
	if err != nil {
		log.Printf("readyz: database: %v", err)
		return 1
	}
	defer db.Close()
	if err := db.Ping(ctx); err != nil {
		log.Printf("readyz: postgres_write: %v", err)
		return 1
	}
	if strings.TrimSpace(cfg.DatabaseReadURL) != "" {
		readDB, err := dbpool.Open(ctx, cfg.DatabaseReadURL, cfg.PGReadMaxConns, cfg.PGReadMinConns)
		if err != nil {
			log.Printf("readyz: database_read: %v", err)
			return 1
		}
		defer readDB.Close()
		if err := readDB.Ping(ctx); err != nil {
			log.Printf("readyz: postgres_read: %v", err)
			return 1
		}
	}
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Printf("readyz: redis url: %v", err)
		return 1
	}
	rdb := redis.NewClient(opt)
	defer rdb.Close()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("readyz: redis: %v", err)
		return 1
	}
	return 0
}
