package login

import (
	"context"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type contextUserKey int

const _userKey contextUserKey = 1

var staticFileRe = regexp.MustCompile(`\.(css|js|gif|jpg|jpeg|png|ico|svg|ttf|eot|woff|woff2)$`)

func Authenticate(b *Builder) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if staticFileRe.MatchString(strings.ToLower(r.URL.Path)) {
				next.ServeHTTP(w, r)
				return
			}

			path := strings.TrimRight(r.URL.Path, "/")
			if strings.HasPrefix(path, "/auth/") &&
				// to redirect to login page
				path != b.loginURL &&
				// below paths need logged-in status
				path != b.logoutURL &&
				path != b.changePasswordURL &&
				path != b.doChangePasswordURL &&
				path != pathTOTPSetup &&
				path != pathTOTPValidate {
				next.ServeHTTP(w, r)
				return
			}

			claims, err := parseUserClaimsFromCookie(r, b.authCookieName, b.secret)
			if err != nil {
				log.Println(err)
				b.setContinueURL(w, r)
				if path == b.loginURL {
					next.ServeHTTP(w, r)
				} else {
					http.Redirect(w, r, b.loginURL, http.StatusFound)
				}
				return
			}

			var user interface{}
			var secureSalt string
			if b.userModel != nil {
				var err error
				user, err = b.findUserByID(claims.UserID)
				if err == nil {
					if claims.Provider == "" {
						if user.(UserPasser).GetPasswordUpdatedAt() != claims.PassUpdatedAt {
							err = errUserPassChanged
						}
						if user.(UserPasser).GetLocked() {
							err = errUserLocked
						}
					} else {
						user.(OAuthUser).SetAvatar(claims.AvatarURL)
					}
				}
				if err != nil {
					log.Println(err)
					switch err {
					case errUserNotFound:
						setFailCodeFlash(w, FailCodeUserNotFound)
					case errUserLocked:
						setFailCodeFlash(w, FailCodeUserLocked)
					case errUserPassChanged:
						setWarnCodeFlash(w, WarnCodePasswordHasBeenChanged)
					default:
						setFailCodeFlash(w, FailCodeSystemError)
					}
					if path == b.logoutURL {
						next.ServeHTTP(w, r)
					} else {
						http.Redirect(w, r, b.logoutURL, http.StatusFound)
					}
					return
				}

				if b.sessionSecureEnabled {
					secureSalt = user.(SessionSecurer).GetSecure()
					_, err := parseBaseClaimsFromCookie(r, b.authSecureCookieName, b.secret+secureSalt)
					if err != nil {
						if path == b.logoutURL {
							next.ServeHTTP(w, r)
						} else {
							http.Redirect(w, r, b.logoutURL, http.StatusFound)
						}
						return
					}
				}
			} else {
				user = claims
			}

			if b.autoExtendSession && time.Now().Sub(claims.IssuedAt.Time).Seconds() > float64(b.sessionMaxAge)/10 {
				claims.RegisteredClaims = b.genBaseSessionClaim(claims.UserID)
				if err := b.setAuthCookiesFromUserClaims(w, claims, secureSalt); err != nil {
					setFailCodeFlash(w, FailCodeSystemError)
					if path == b.logoutURL {
						next.ServeHTTP(w, r)
					} else {
						http.Redirect(w, r, b.logoutURL, http.StatusFound)
					}
					return
				}
			}

			r = r.WithContext(context.WithValue(r.Context(), _userKey, user))

			if claims.Provider == "" && b.totpEnabled && !claims.TOTPValidated {
				if path == b.loginURL {
					next.ServeHTTP(w, r)
					return
				}
				if !user.(UserPasser).GetIsTOTPSetup() {
					if path == pathTOTPSetup {
						next.ServeHTTP(w, r)
						return
					}
					http.Redirect(w, r, pathTOTPSetup, http.StatusFound)
					return
				}
				if path == pathTOTPValidate {
					next.ServeHTTP(w, r)
					return
				}
				http.Redirect(w, r, pathTOTPValidate, http.StatusFound)
				return
			}

			if claims.TOTPValidated || claims.Provider != "" {
				if path == pathTOTPSetup || path == pathTOTPValidate {
					http.Redirect(w, r, b.homeURL, http.StatusFound)
					return
				}
			}

			if path == b.loginURL {
				http.Redirect(w, r, b.homeURL, http.StatusFound)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func GetCurrentUser(r *http.Request) (u interface{}) {
	return r.Context().Value(_userKey)
}
